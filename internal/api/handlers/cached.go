package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/cache"
	"github.com/sadewadee/google-scraper/internal/domain"
)

// CachedStatsHandler wraps StatsHandler with Redis cache
type CachedStatsHandler struct {
	stats StatsServiceInterface
	cache cache.Cache
}

// NewCachedStatsHandler creates a new CachedStatsHandler
func NewCachedStatsHandler(stats StatsServiceInterface, c cache.Cache) *CachedStatsHandler {
	return &CachedStatsHandler{
		stats: stats,
		cache: c,
	}
}

// GetDashboardStats handles GET /api/v2/stats with caching
func (h *CachedStatsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()
	cacheKey := cache.KeyPrefixDashboardStats

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedStats] Cache HIT for stats")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedStats] Cache MISS for stats")
	stats, err := h.stats.GetStats(ctx)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	// Serialize and cache
	data, err := json.Marshal(stats)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLStats); cacheErr != nil {
			log.Printf("[CachedStats] Failed to cache stats: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, stats)
}

// CachedJobHandler wraps JobHandler with Redis cache
type CachedJobHandler struct {
	jobs    JobServiceInterface
	results ResultServiceInterface
	cache   cache.Cache
}

// NewCachedJobHandler creates a new CachedJobHandler
func NewCachedJobHandler(jobs JobServiceInterface, results ResultServiceInterface, c cache.Cache) *CachedJobHandler {
	return &CachedJobHandler{
		jobs:    jobs,
		results: results,
		cache:   c,
	}
}

// List handles GET /api/v2/jobs with caching
func (h *CachedJobHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	status := r.URL.Query().Get("status")

	// Build cache key
	cacheKey := fmt.Sprintf("%s:list:page=%d:perPage=%d:status=%s",
		cache.KeyPrefixDashboardJobs, page, perPage, status)

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedJobs] Cache HIT for jobs list")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedJobs] Cache MISS for jobs list")
	params := domain.JobListParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	if status != "" {
		s := domain.JobStatus(status)
		params.Status = &s
	}

	jobs, total, err := h.jobs.List(ctx, params)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to list jobs: "+err.Error())
		return
	}

	response := NewPaginatedResponse(jobs, total, page, perPage)

	// Serialize and cache
	data, err := json.Marshal(response)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLJobsList); cacheErr != nil {
			log.Printf("[CachedJobs] Failed to cache jobs list: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, response)
}

// GetByID handles GET /api/v2/jobs/{id} with caching
func (h *CachedJobHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	ctx := r.Context()
	cacheKey := fmt.Sprintf("%s:%s", cache.KeyPrefixDashboardJobs, id.String())

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedJobs] Cache HIT for job %s", id)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedJobs] Cache MISS for job %s", id)
	job, err := h.jobs.GetByID(ctx, id)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to retrieve job: "+err.Error())
		return
	}

	if job == nil {
		RenderError(w, http.StatusNotFound, "Job not found")
		return
	}

	// Serialize and cache
	data, err := json.Marshal(job)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLJobDetail); cacheErr != nil {
			log.Printf("[CachedJobs] Failed to cache job: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, job)
}

// GetResults handles GET /api/v2/jobs/{id}/results with caching
func (h *CachedJobHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	ctx := r.Context()

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Build cache key
	cacheKey := fmt.Sprintf("%s:%s:page=%d:perPage=%d",
		cache.KeyPrefixDashboardResults, id.String(), page, perPage)

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedJobs] Cache HIT for results job %s", id)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedJobs] Cache MISS for results job %s", id)
	offset := (page - 1) * perPage

	results, total, err := h.results.ListByJobID(ctx, id, perPage, offset)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get results: "+err.Error())
		return
	}

	// Parse JSON data
	var parsedResults []json.RawMessage
	for _, data := range results {
		parsedResults = append(parsedResults, json.RawMessage(data))
	}

	response := NewPaginatedResponse(parsedResults, total, page, perPage)

	// Serialize and cache
	data, err := json.Marshal(response)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLResults); cacheErr != nil {
			log.Printf("[CachedJobs] Failed to cache results: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, response)
}

// GetStats handles GET /api/v2/jobs/stats with caching
func (h *CachedJobHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()
	cacheKey := cache.KeyPrefixDashboardJobs + ":stats"

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedJobs] Cache HIT for job stats")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedJobs] Cache MISS for job stats")
	stats, err := h.jobs.GetStats(ctx)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	// Serialize and cache
	data, err := json.Marshal(stats)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLStats); cacheErr != nil {
			log.Printf("[CachedJobs] Failed to cache job stats: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, stats)
}

// InvalidateJobCache invalidates all cache related to a job
func (h *CachedJobHandler) InvalidateJobCache(ctx context.Context, jobID uuid.UUID) error {
	// Invalidate job detail
	if err := h.cache.Delete(ctx, fmt.Sprintf("%s:%s", cache.KeyPrefixDashboardJobs, jobID.String())); err != nil {
		log.Printf("[CachedJobs] Failed to invalidate job cache: %v", err)
	}

	// Invalidate job list (all pages)
	if err := h.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardJobs+":list:*"); err != nil {
		log.Printf("[CachedJobs] Failed to invalidate job list cache: %v", err)
	}

	// Invalidate results for this job
	if err := h.cache.DeleteByPattern(ctx, fmt.Sprintf("%s:%s:*", cache.KeyPrefixDashboardResults, jobID.String())); err != nil {
		log.Printf("[CachedJobs] Failed to invalidate results cache: %v", err)
	}

	// Invalidate stats
	if err := h.cache.Delete(ctx, cache.KeyPrefixDashboardStats); err != nil {
		log.Printf("[CachedJobs] Failed to invalidate stats cache: %v", err)
	}

	if err := h.cache.Delete(ctx, cache.KeyPrefixDashboardJobs+":stats"); err != nil {
		log.Printf("[CachedJobs] Failed to invalidate job stats cache: %v", err)
	}

	return nil
}

// CachedResultHandler wraps ResultHandler with Redis cache
type CachedResultHandler struct {
	results GlobalResultServiceInterface
	cache   cache.Cache
}

// NewCachedResultHandler creates a new CachedResultHandler
func NewCachedResultHandler(results GlobalResultServiceInterface, c cache.Cache) *CachedResultHandler {
	return &CachedResultHandler{
		results: results,
		cache:   c,
	}
}

// List handles GET /api/v2/results with caching
func (h *CachedResultHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Build cache key
	cacheKey := fmt.Sprintf("%s:all:page=%d:perPage=%d",
		cache.KeyPrefixDashboardResults, page, perPage)

	// Try cache first
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("[CachedResults] Cache HIT for results list")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Cache miss - fetch from service
	log.Printf("[CachedResults] Cache MISS for results list")
	offset := (page - 1) * perPage

	results, total, err := h.results.ListAll(ctx, perPage, offset)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get results: "+err.Error())
		return
	}

	// Parse JSON data
	var parsedResults []json.RawMessage
	for _, data := range results {
		parsedResults = append(parsedResults, json.RawMessage(data))
	}

	response := NewPaginatedResponse(parsedResults, total, page, perPage)

	// Serialize and cache
	data, err := json.Marshal(response)
	if err == nil {
		if cacheErr := h.cache.Set(ctx, cacheKey, data, cache.TTLResults); cacheErr != nil {
			log.Printf("[CachedResults] Failed to cache results: %v", cacheErr)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, response)
}

// InvalidateResultsCache invalidates all results cache
func (h *CachedResultHandler) InvalidateResultsCache(ctx context.Context) error {
	if err := h.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardResults+":*"); err != nil {
		return err
	}
	return h.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardSearch+":*")
}

// CacheInvalidator provides cache invalidation methods for use by other components
type CacheInvalidator struct {
	cache cache.Cache
}

// NewCacheInvalidator creates a new CacheInvalidator
func NewCacheInvalidator(c cache.Cache) *CacheInvalidator {
	return &CacheInvalidator{cache: c}
}

// InvalidateOnNewResults invalidates caches when new results are added
func (ci *CacheInvalidator) InvalidateOnNewResults(ctx context.Context, jobID uuid.UUID) error {
	// Invalidate results for this job
	if err := ci.cache.DeleteByPattern(ctx, fmt.Sprintf("%s:%s:*", cache.KeyPrefixDashboardResults, jobID.String())); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate job results: %v", err)
	}

	// Invalidate global results
	if err := ci.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardResults+":all:*"); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate all results: %v", err)
	}

	// Invalidate search cache
	if err := ci.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardSearch+":*"); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate search cache: %v", err)
	}

	// Invalidate stats
	if err := ci.cache.Delete(ctx, cache.KeyPrefixDashboardStats); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate stats: %v", err)
	}

	return nil
}

// InvalidateOnJobStatusChange invalidates caches when job status changes
func (ci *CacheInvalidator) InvalidateOnJobStatusChange(ctx context.Context, jobID uuid.UUID) error {
	// Invalidate job detail
	if err := ci.cache.Delete(ctx, fmt.Sprintf("%s:%s", cache.KeyPrefixDashboardJobs, jobID.String())); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate job: %v", err)
	}

	// Invalidate job list
	if err := ci.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardJobs+":list:*"); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate job list: %v", err)
	}

	// Invalidate job stats
	if err := ci.cache.Delete(ctx, cache.KeyPrefixDashboardJobs+":stats"); err != nil {
		log.Printf("[CacheInvalidator] Failed to invalidate job stats: %v", err)
	}

	return nil
}

// CachedResponseWriter captures the response for caching
type CachedResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// NewCachedResponseWriter creates a new CachedResponseWriter
func NewCachedResponseWriter(w http.ResponseWriter) *CachedResponseWriter {
	return &CachedResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

// WriteHeader captures the status code
func (crw *CachedResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (crw *CachedResponseWriter) Write(b []byte) (int, error) {
	crw.body.Write(b)
	return crw.ResponseWriter.Write(b)
}

// StatusCode returns the captured status code
func (crw *CachedResponseWriter) StatusCode() int {
	return crw.statusCode
}

// Body returns the captured response body
func (crw *CachedResponseWriter) Body() []byte {
	return crw.body.Bytes()
}
