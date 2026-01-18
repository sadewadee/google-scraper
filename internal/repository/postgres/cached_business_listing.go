package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sadewadee/google-scraper/internal/cache"
	"github.com/sadewadee/google-scraper/internal/domain"
)

// CachedBusinessListingRepository wraps BusinessListingRepository with Redis caching
// for expensive COUNT and aggregation queries to prevent 504 timeouts
type CachedBusinessListingRepository struct {
	repo      *BusinessListingRepository
	cache     cache.Cache
	db        *sql.DB
	hasCache  bool // true if cache is a real implementation (not NoOpCache)
}

// Cache TTLs
const (
	countCacheTTL    = 60 * time.Second  // Total count cache (short TTL for freshness)
	statsCacheTTL    = 120 * time.Second // Stats cache (more expensive, longer TTL)
	listCacheTTL     = 30 * time.Second  // List results cache (shortest TTL)
	categoryCacheTTL = 300 * time.Second // Categories/cities cache (rarely changes)
)

// Cache key prefixes
const (
	keyPrefixCount      = "bl:count:"
	keyPrefixStats      = "bl:stats"
	keyPrefixList       = "bl:list:"
	keyPrefixCategories = "bl:categories:"
	keyPrefixCities     = "bl:cities:"
	keyPrefixJobCount   = "bl:jobcount:"
	keyTotalApprox      = "bl:total:approx"
)

// NewCachedBusinessListingRepository creates a new cached repository
func NewCachedBusinessListingRepository(db *sql.DB, c cache.Cache) *CachedBusinessListingRepository {
	// Check if this is a real cache implementation or a NoOpCache
	_, isNoOp := c.(*cache.NoOpCache)
	return &CachedBusinessListingRepository{
		repo:     NewBusinessListingRepository(db),
		cache:    c,
		db:       db,
		hasCache: !isNoOp,
	}
}

// filterCacheKey generates a unique cache key based on filter parameters
func filterCacheKey(filter domain.BusinessListingFilter) string {
	// Create a deterministic representation of the filter
	data := fmt.Sprintf("%v|%s|%s|%s|%s|%v|%v|%s",
		filter.JobID, filter.Search, filter.Category, filter.City, filter.Country,
		filter.MinRating, filter.HasEmail, filter.EmailStatus)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter key
}

// List retrieves business listings with caching for the COUNT query
func (r *CachedBusinessListingRepository) List(ctx context.Context, filter domain.BusinessListingFilter) ([]*domain.BusinessListing, int, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 25
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	// Generate cache key for count
	countKey := keyPrefixCount + filterCacheKey(filter)

	// Try to get count from cache first
	var total int
	var countCached bool

	if cached, err := r.cache.Get(ctx, countKey); err == nil {
		if err := json.Unmarshal(cached, &total); err == nil {
			countCached = true
		}
	}

	if !countCached {
		// Check if this is a simple query (no filters) - use approximate count
		if r.isSimpleQuery(filter) {
			approxCount, err := r.getApproximateCount(ctx)
			if err == nil && approxCount > 0 {
				total = approxCount
				countCached = true
				// Cache the approximate count with shorter TTL
				if data, err := json.Marshal(total); err == nil {
					_ = r.cache.Set(ctx, countKey, data, countCacheTTL/2)
				}
			}
		}

		if !countCached {
			// Fall back to exact count (expensive)
			listings, exactTotal, err := r.repo.List(ctx, filter)
			if err != nil {
				return nil, 0, err
			}
			total = exactTotal

			// Cache the count
			if data, err := json.Marshal(total); err == nil {
				_ = r.cache.Set(ctx, countKey, data, countCacheTTL)
			}

			return listings, total, nil
		}
	}

	// Now get the actual listing data (still need to query for the page)
	listings, _, err := r.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return listings, total, nil
}

// isSimpleQuery checks if the filter has no search/filter conditions
func (r *CachedBusinessListingRepository) isSimpleQuery(filter domain.BusinessListingFilter) bool {
	return filter.JobID == nil &&
		filter.Search == "" &&
		filter.Category == "" &&
		filter.City == "" &&
		filter.Country == "" &&
		filter.MinRating == nil &&
		filter.HasEmail == nil &&
		filter.EmailStatus == ""
}

// getApproximateCount uses PostgreSQL's pg_class.reltuples for fast count estimation
func (r *CachedBusinessListingRepository) getApproximateCount(ctx context.Context) (int, error) {
	// Try cache first
	if cached, err := r.cache.Get(ctx, keyTotalApprox); err == nil {
		var count int
		if err := json.Unmarshal(cached, &count); err == nil {
			return count, nil
		}
	}

	// Use pg_class for fast approximate count
	query := `SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE relname = 'business_listings'`
	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}

	// If reltuples is 0 (VACUUM/ANALYZE never ran), do a quick count
	if count == 0 {
		countQuery := `SELECT COUNT(*) FROM business_listings`
		if err := r.db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
			return 0, err
		}
	}

	// Cache the approximate count
	if data, err := json.Marshal(count); err == nil {
		_ = r.cache.Set(ctx, keyTotalApprox, data, countCacheTTL)
	}

	return count, nil
}

// ListByJobID retrieves business listings for a specific job with caching
func (r *CachedBusinessListingRepository) ListByJobID(ctx context.Context, jobID string, limit, offset int) ([]*domain.BusinessListing, int, error) {
	// Try to get count from cache
	countKey := keyPrefixJobCount + jobID
	var total int
	var countCached bool

	if cached, err := r.cache.Get(ctx, countKey); err == nil {
		if err := json.Unmarshal(cached, &total); err == nil {
			countCached = true
		}
	}

	if !countCached {
		// Get exact count from repo
		listings, exactTotal, err := r.repo.ListByJobID(ctx, jobID, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		total = exactTotal

		// Cache the count
		if data, err := json.Marshal(total); err == nil {
			_ = r.cache.Set(ctx, countKey, data, countCacheTTL)
		}

		return listings, total, nil
	}

	// Get listings with cached count
	listings, _, err := r.repo.ListByJobID(ctx, jobID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return listings, total, nil
}

// Stats retrieves aggregate statistics with caching
func (r *CachedBusinessListingRepository) Stats(ctx context.Context) (*domain.BusinessListingStats, error) {
	// Try cache first
	if cached, err := r.cache.Get(ctx, keyPrefixStats); err == nil {
		var stats domain.BusinessListingStats
		if err := json.Unmarshal(cached, &stats); err == nil {
			return &stats, nil
		}
	}

	// Get fresh stats
	stats, err := r.repo.Stats(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the stats
	if data, err := json.Marshal(stats); err == nil {
		_ = r.cache.Set(ctx, keyPrefixStats, data, statsCacheTTL)
	}

	return stats, nil
}

// GetCategories returns distinct categories with caching
func (r *CachedBusinessListingRepository) GetCategories(ctx context.Context, limit int) ([]string, error) {
	cacheKey := fmt.Sprintf("%s%d", keyPrefixCategories, limit)

	// Try cache first
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
		var categories []string
		if err := json.Unmarshal(cached, &categories); err == nil {
			return categories, nil
		}
	}

	// Get fresh categories
	categories, err := r.repo.GetCategories(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Cache the categories
	if data, err := json.Marshal(categories); err == nil {
		_ = r.cache.Set(ctx, cacheKey, data, categoryCacheTTL)
	}

	return categories, nil
}

// GetCities returns distinct cities with caching
func (r *CachedBusinessListingRepository) GetCities(ctx context.Context, limit int) ([]string, error) {
	cacheKey := fmt.Sprintf("%s%d", keyPrefixCities, limit)

	// Try cache first
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
		var cities []string
		if err := json.Unmarshal(cached, &cities); err == nil {
			return cities, nil
		}
	}

	// Get fresh cities
	cities, err := r.repo.GetCities(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Cache the cities
	if data, err := json.Marshal(cities); err == nil {
		_ = r.cache.Set(ctx, cacheKey, data, categoryCacheTTL)
	}

	return cities, nil
}

// GetByID retrieves a single business listing by ID (no caching, fast)
func (r *CachedBusinessListingRepository) GetByID(ctx context.Context, id int64) (*domain.BusinessListing, error) {
	return r.repo.GetByID(ctx, id)
}

// Stream streams business listings for export (no caching)
func (r *CachedBusinessListingRepository) Stream(ctx context.Context, filter domain.BusinessListingFilter, fn func(listing *domain.BusinessListing) error) error {
	return r.repo.Stream(ctx, filter, fn)
}

// StreamByJobID streams business listings for a specific job (no caching)
func (r *CachedBusinessListingRepository) StreamByJobID(ctx context.Context, jobID string, fn func(listing *domain.BusinessListing) error) error {
	return r.repo.StreamByJobID(ctx, jobID, fn)
}

// CountByJobID counts business listings for a job
func (r *CachedBusinessListingRepository) CountByJobID(ctx context.Context, jobID string) (int, error) {
	countKey := keyPrefixJobCount + jobID

	// Try cache first
	if cached, err := r.cache.Get(ctx, countKey); err == nil {
		var count int
		if err := json.Unmarshal(cached, &count); err == nil {
			return count, nil
		}
	}

	// Get fresh count
	count, err := r.repo.CountByJobID(ctx, jobID)
	if err != nil {
		return 0, err
	}

	// Cache the count
	if data, err := json.Marshal(count); err == nil {
		_ = r.cache.Set(ctx, countKey, data, countCacheTTL)
	}

	return count, nil
}

// InvalidateJobCache invalidates cache for a specific job
// Call this when job results are updated
func (r *CachedBusinessListingRepository) InvalidateJobCache(ctx context.Context, jobID string) error {
	countKey := keyPrefixJobCount + jobID
	if err := r.cache.Delete(ctx, countKey); err != nil {
		log.Printf("[CachedBusinessListingRepo] Failed to invalidate job cache %s: %v", jobID, err)
	}
	return nil
}

// InvalidateAllCache invalidates all business listing caches
// Call this after bulk operations
func (r *CachedBusinessListingRepository) InvalidateAllCache(ctx context.Context) error {
	patterns := []string{
		keyPrefixCount + "*",
		keyPrefixJobCount + "*",
		keyPrefixStats,
		keyTotalApprox,
	}

	for _, pattern := range patterns {
		if err := r.cache.DeleteByPattern(ctx, pattern); err != nil {
			log.Printf("[CachedBusinessListingRepo] Failed to invalidate pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// PreloadCache preloads common cache entries on startup
// This is the "preload" strategy the user asked for
func (r *CachedBusinessListingRepository) PreloadCache(ctx context.Context) error {
	log.Println("[CachedBusinessListingRepo] Preloading cache...")

	// Preload approximate total count
	count, err := r.getApproximateCount(ctx)
	if err != nil {
		log.Printf("[CachedBusinessListingRepo] Failed to preload total count: %v", err)
	} else {
		log.Printf("[CachedBusinessListingRepo] Preloaded total count: %d", count)
	}

	// Preload stats (runs in background, don't block startup)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stats, err := r.Stats(bgCtx)
		if err != nil {
			log.Printf("[CachedBusinessListingRepo] Failed to preload stats: %v", err)
		} else {
			log.Printf("[CachedBusinessListingRepo] Preloaded stats: %d listings, %d emails",
				stats.TotalListings, stats.TotalEmails)
		}
	}()

	// Preload categories (runs in background)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		categories, err := r.GetCategories(bgCtx, 50)
		if err != nil {
			log.Printf("[CachedBusinessListingRepo] Failed to preload categories: %v", err)
		} else {
			log.Printf("[CachedBusinessListingRepo] Preloaded %d categories", len(categories))
		}
	}()

	return nil
}

// Verify interface compliance at compile time
var _ domain.BusinessListingRepository = (*CachedBusinessListingRepository)(nil)
