package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// BusinessListingRepository provides access to business_listings
type BusinessListingRepository struct {
	db *sql.DB
}

// NewBusinessListingRepository creates a new repository
func NewBusinessListingRepository(db *sql.DB) *BusinessListingRepository {
	return &BusinessListingRepository{db: db}
}

// escapeLikePattern escapes LIKE metacharacters in search strings
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// validEmailStatuses contains all valid email validation statuses
var validEmailStatuses = map[string]bool{
	"pending":       true,
	"local_valid":   true,
	"local_invalid": true,
	"api_valid":     true,
	"api_invalid":   true,
	"api_error":     true,
	"api_skipped":   true,
}

// filterResult holds the result of building filter clauses
type filterResult struct {
	whereClause  string
	havingClause string
	args         []interface{}
	nextArgNum   int
}

// buildFilterClauses builds WHERE and HAVING clauses from filter parameters
func buildFilterClauses(filter domain.BusinessListingFilter, startArgNum int) filterResult {
	var conditions []string
	var args []interface{}
	argNum := startArgNum

	if filter.JobID != nil {
		conditions = append(conditions, fmt.Sprintf("bl.job_id = $%d", argNum))
		args = append(args, filter.JobID.String())
		argNum++
	}

	if filter.Search != "" {
		searchPattern := "%" + escapeLikePattern(filter.Search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(bl.title ILIKE $%d OR bl.address ILIKE $%d OR bl.phone ILIKE $%d OR bl.category ILIKE $%d)",
			argNum, argNum, argNum, argNum,
		))
		args = append(args, searchPattern)
		argNum++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("bl.category = $%d", argNum))
		args = append(args, filter.Category)
		argNum++
	}

	if filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("bl.address_city = $%d", argNum))
		args = append(args, filter.City)
		argNum++
	}

	if filter.Country != "" {
		conditions = append(conditions, fmt.Sprintf("bl.address_country = $%d", argNum))
		args = append(args, filter.Country)
		argNum++
	}

	if filter.MinRating != nil {
		conditions = append(conditions, fmt.Sprintf("bl.review_rating >= $%d", argNum))
		args = append(args, *filter.MinRating)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build HAVING clause for email filters (applied after GROUP BY)
	var havingConditions []string
	if filter.HasEmail != nil {
		if *filter.HasEmail {
			// Has at least one email
			havingConditions = append(havingConditions, "COUNT(DISTINCT e.id) > 0")
		} else {
			// Has no emails
			havingConditions = append(havingConditions, "COUNT(DISTINCT e.id) = 0")
		}
	}

	if filter.EmailStatus != "" && validEmailStatuses[filter.EmailStatus] {
		havingConditions = append(havingConditions, fmt.Sprintf(
			"COUNT(DISTINCT e.id) FILTER (WHERE e.validation_status = $%d) > 0", argNum))
		args = append(args, filter.EmailStatus)
		argNum++
	}

	havingClause := ""
	if len(havingConditions) > 0 {
		havingClause = "HAVING " + strings.Join(havingConditions, " AND ")
	}

	return filterResult{
		whereClause:  whereClause,
		havingClause: havingClause,
		args:         args,
		nextArgNum:   argNum,
	}
}

// scanListing is a helper to scan a business listing row into a domain.BusinessListing
func (r *BusinessListingRepository) scanListing(rows *sql.Rows) (*domain.BusinessListing, error) {
	var bl domain.BusinessListing
	var jobID, placeID, cid, category, address, phone, website sql.NullString
	var addressCity, addressCountry, status, priceRange, link sql.NullString
	var latitude, longitude, reviewRating sql.NullFloat64
	var categories []byte
	var emailsInfoJSON []byte
	var emailsArray []byte

	err := rows.Scan(
		&bl.ID, &bl.ResultID, &jobID, &placeID, &cid,
		&bl.Title, &category, &categories, &address, &phone,
		&website, &latitude, &longitude, &addressCity, &addressCountry,
		&bl.ReviewCount, &reviewRating, &status, &priceRange, &link,
		&bl.CreatedAt,
		&emailsInfoJSON, &emailsArray,
		&bl.ValidEmailCount, &bl.TotalEmailCount,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if jobID.Valid {
		bl.JobID = &jobID.String
	}
	if placeID.Valid {
		bl.PlaceID = &placeID.String
	}
	if cid.Valid {
		bl.CID = &cid.String
	}
	if category.Valid {
		bl.Category = &category.String
	}
	if address.Valid {
		bl.Address = &address.String
	}
	if phone.Valid {
		bl.Phone = &phone.String
	}
	if website.Valid {
		bl.Website = &website.String
	}
	if latitude.Valid {
		bl.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		bl.Longitude = &longitude.Float64
	}
	if addressCity.Valid {
		bl.AddressCity = &addressCity.String
	}
	if addressCountry.Valid {
		bl.AddressCountry = &addressCountry.String
	}
	if reviewRating.Valid {
		bl.ReviewRating = &reviewRating.Float64
	}
	if status.Valid {
		bl.Status = &status.String
	}
	if priceRange.Valid {
		bl.PriceRange = &priceRange.String
	}
	if link.Valid {
		bl.Link = &link.String
	}

	// Parse categories array
	if len(categories) > 0 {
		if err := json.Unmarshal(categories, &bl.Categories); err != nil {
			log.Printf("[BusinessListingRepository] Warning: failed to unmarshal categories for listing %d: %v", bl.ID, err)
		}
	}

	// Parse emails info JSON
	if len(emailsInfoJSON) > 0 {
		if err := json.Unmarshal(emailsInfoJSON, &bl.EmailsWithInfo); err != nil {
			log.Printf("[BusinessListingRepository] Warning: failed to unmarshal emails_info for listing %d: %v", bl.ID, err)
		}
	}

	// Parse emails array
	if len(emailsArray) > 0 {
		if err := json.Unmarshal(emailsArray, &bl.Emails); err != nil {
			log.Printf("[BusinessListingRepository] Warning: failed to unmarshal emails for listing %d: %v", bl.ID, err)
		}
	}

	return &bl, nil
}

// baseSelectQuery returns the common SELECT query for business listings
func baseSelectQuery() string {
	return `
		SELECT
			bl.id, bl.result_id, bl.job_id, bl.place_id, bl.cid,
			bl.title, bl.category, COALESCE(array_to_json(bl.categories), '[]'::json) AS categories, bl.address, bl.phone,
			bl.website, bl.latitude, bl.longitude, bl.address_city, bl.address_country,
			bl.review_count, bl.review_rating, bl.status, bl.price_range, bl.link,
			bl.created_at,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'email', e.email,
						'status', e.validation_status,
						'score', e.api_score,
						'is_acceptable', e.is_acceptable
					)
				) FILTER (WHERE e.id IS NOT NULL),
				'[]'::jsonb
			) AS emails_info,
			COALESCE(array_to_json(array_agg(DISTINCT e.email) FILTER (WHERE e.id IS NOT NULL)), '[]'::json) AS emails,
			COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) AS valid_email_count,
			COUNT(DISTINCT e.id) AS total_email_count
		FROM business_listings bl
		LEFT JOIN business_emails be ON be.business_listing_id = bl.id
		LEFT JOIN emails e ON e.id = be.email_id
	`
}

// List retrieves business listings with filters
func (r *BusinessListingRepository) List(ctx context.Context, filter domain.BusinessListingFilter) ([]*domain.BusinessListing, int, error) {
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

	// Build filter clauses
	fr := buildFilterClauses(filter, 1)
	whereClause := fr.whereClause
	havingClause := fr.havingClause
	args := fr.args
	argNum := fr.nextArgNum

	// Validate sort column
	validSortColumns := map[string]string{
		"created_at":    "bl.created_at",
		"review_rating": "bl.review_rating",
		"review_count":  "bl.review_count",
		"title":         "bl.title",
	}
	sortColumn, ok := validSortColumns[filter.SortBy]
	if !ok {
		sortColumn = "bl.created_at"
	}

	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	// Count total - need to use subquery when HAVING is present
	var countQuery string
	if havingClause != "" {
		// When using HAVING, we need to count the grouped results
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM (
				SELECT bl.id
				FROM business_listings bl
				LEFT JOIN business_emails be ON be.business_listing_id = bl.id
				LEFT JOIN emails e ON e.id = be.email_id
				%s
				GROUP BY bl.id
				%s
			) AS filtered
		`, whereClause, havingClause)
	} else {
		countQuery = fmt.Sprintf(`
			SELECT COUNT(DISTINCT bl.id)
			FROM business_listings bl
			%s
		`, whereClause)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Main query with email aggregation
	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`%s %s
		GROUP BY bl.id
		%s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, baseSelectQuery(), whereClause, havingClause, sortColumn, sortOrder, argNum, argNum+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var listings []*domain.BusinessListing
	for rows.Next() {
		bl, err := r.scanListing(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}
		listings = append(listings, bl)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return listings, total, nil
}

// ListByJobID retrieves business listings for a specific job with pagination
func (r *BusinessListingRepository) ListByJobID(ctx context.Context, jobID string, limit, offset int) ([]*domain.BusinessListing, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM business_listings WHERE job_id = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, jobID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`%s WHERE bl.job_id = $1
		GROUP BY bl.id
		ORDER BY bl.created_at DESC
		LIMIT $2 OFFSET $3
	`, baseSelectQuery())

	rows, err := r.db.QueryContext(ctx, query, jobID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list by job id query failed: %w", err)
	}
	defer rows.Close()

	var listings []*domain.BusinessListing
	for rows.Next() {
		bl, err := r.scanListing(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}
		listings = append(listings, bl)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return listings, total, nil
}

// GetByID retrieves a single business listing by ID
func (r *BusinessListingRepository) GetByID(ctx context.Context, id int64) (*domain.BusinessListing, error) {
	query := fmt.Sprintf(`%s WHERE bl.id = $1 GROUP BY bl.id`, baseSelectQuery())

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get by id query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	return r.scanListing(rows)
}

// GetCategories returns distinct categories
func (r *BusinessListingRepository) GetCategories(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT category, COUNT(*) as cnt
		FROM business_listings
		WHERE category IS NOT NULL AND category != ''
		GROUP BY category
		ORDER BY cnt DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get categories failed: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("scan category failed: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return categories, nil
}

// GetCities returns distinct cities
func (r *BusinessListingRepository) GetCities(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT address_city, COUNT(*) as cnt
		FROM business_listings
		WHERE address_city IS NOT NULL AND address_city != ''
		GROUP BY address_city
		ORDER BY cnt DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get cities failed: %w", err)
	}
	defer rows.Close()

	var cities []string
	for rows.Next() {
		var city string
		var count int
		if err := rows.Scan(&city, &count); err != nil {
			return nil, fmt.Errorf("scan city failed: %w", err)
		}
		cities = append(cities, city)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return cities, nil
}

// Stats returns aggregate statistics
func (r *BusinessListingRepository) Stats(ctx context.Context) (*domain.BusinessListingStats, error) {
	query := `
		SELECT
			COUNT(*) as total_listings,
			COUNT(DISTINCT bl.job_id) as total_jobs,
			COUNT(DISTINCT e.id) as total_emails,
			COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) as valid_emails,
			AVG(bl.review_rating) FILTER (WHERE bl.review_rating IS NOT NULL) as avg_rating,
			COUNT(*) FILTER (WHERE bl.phone IS NOT NULL AND bl.phone != '') as with_phone,
			COUNT(*) FILTER (WHERE bl.website IS NOT NULL AND bl.website != '') as with_website
		FROM business_listings bl
		LEFT JOIN business_emails be ON be.business_listing_id = bl.id
		LEFT JOIN emails e ON e.id = be.email_id
	`

	var stats domain.BusinessListingStats
	var avgRating sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalListings, &stats.TotalJobs, &stats.TotalEmails, &stats.ValidEmails,
		&avgRating, &stats.WithPhone, &stats.WithWebsite,
	)
	if err != nil {
		return nil, fmt.Errorf("stats query failed: %w", err)
	}

	if avgRating.Valid {
		stats.AvgRating = &avgRating.Float64
	}

	return &stats, nil
}

// Stream streams business listings for export (memory efficient)
func (r *BusinessListingRepository) Stream(ctx context.Context, filter domain.BusinessListingFilter, fn func(listing *domain.BusinessListing) error) error {
	// Build filter clauses
	fr := buildFilterClauses(filter, 1)

	query := fmt.Sprintf(`%s %s GROUP BY bl.id %s ORDER BY bl.created_at DESC`,
		baseSelectQuery(), fr.whereClause, fr.havingClause)

	rows, err := r.db.QueryContext(ctx, query, fr.args...)
	if err != nil {
		return fmt.Errorf("stream query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		bl, err := r.scanListing(rows)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		if err := fn(bl); err != nil {
			return err
		}
	}

	return rows.Err()
}

// StreamByJobID streams business listings for a specific job (memory efficient)
func (r *BusinessListingRepository) StreamByJobID(ctx context.Context, jobID string, fn func(listing *domain.BusinessListing) error) error {
	query := fmt.Sprintf(`%s WHERE bl.job_id = $1 GROUP BY bl.id ORDER BY bl.created_at DESC`, baseSelectQuery())

	rows, err := r.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("stream by job id query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		bl, err := r.scanListing(rows)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		if err := fn(bl); err != nil {
			return err
		}
	}

	return rows.Err()
}

// CountByJobID counts business listings for a job
func (r *BusinessListingRepository) CountByJobID(ctx context.Context, jobID string) (int, error) {
	query := `SELECT COUNT(*) FROM business_listings WHERE job_id = $1`
	var count int
	if err := r.db.QueryRowContext(ctx, query, jobID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count by job id failed: %w", err)
	}
	return count, nil
}

// Verify interface compliance at compile time
var _ domain.BusinessListingRepository = (*BusinessListingRepository)(nil)
