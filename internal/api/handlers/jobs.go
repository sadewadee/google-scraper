package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/sadewadee/google-scraper/gmaps"
	"github.com/sadewadee/google-scraper/internal/cache"
	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/service"
)

// JobServiceInterface defines the job service methods
type JobServiceInterface interface {
	Create(ctx context.Context, req *domain.CreateJobRequest) (*domain.Job, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	List(ctx context.Context, params domain.JobListParams) ([]*domain.Job, int, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Pause(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	Resume(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	Cancel(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, progress domain.JobProgress) error
	GetStats(ctx context.Context) (*domain.JobStats, error)
}

// ResultServiceInterface defines the result service methods
type ResultServiceInterface interface {
	CreateBatch(ctx context.Context, jobID uuid.UUID, data [][]byte) error
	ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error)
	StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error
	CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error)
}

// JobHandler handles job-related HTTP requests
type JobHandler struct {
	jobs    JobServiceInterface
	results ResultServiceInterface
	cache   cache.Cache
}

// MaxResultBatchSize is the maximum size of a result batch (10MB)
const MaxResultBatchSize = 10 << 20

// downloadTimeout is the timeout for large download operations (5 minutes)
const downloadTimeout = 5 * time.Minute

// SubmitResults handles POST /api/v2/jobs/{id}/results
func (h *JobHandler) SubmitResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		log.Printf("[SubmitResults] Invalid job ID: %v", err)
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	log.Printf("[SubmitResults] Receiving results for job %s", id)

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, MaxResultBatchSize)

	var batch domain.ResultBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		log.Printf("[SubmitResults] Failed to decode request body: %v", err)
		RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("[SubmitResults] Job %s: Received batch with %d results", id, len(batch.Data))

	if batch.JobID != uuid.Nil && batch.JobID != id {
		log.Printf("[SubmitResults] Job ID mismatch: URL=%s, Body=%s", id, batch.JobID)
		RenderError(w, http.StatusBadRequest, "Job ID mismatch")
		return
	}

	if len(batch.Data) == 0 {
		log.Printf("[SubmitResults] Job %s: Empty batch, returning 204", id)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.results.CreateBatch(r.Context(), id, batch.Data); err != nil {
		log.Printf("[SubmitResults] Job %s: CreateBatch FAILED: %v", id, err)
		RenderError(w, http.StatusInternalServerError, "Failed to save results")
		return
	}

	log.Printf("[SubmitResults] Job %s: Successfully saved %d results to database", id, len(batch.Data))

	// Update scraped_places counter from actual database count
	totalResults, countErr := h.results.CountByJobID(r.Context(), id)
	if countErr != nil {
		log.Printf("[SubmitResults] Job %s: WARNING - failed to count results: %v", id, countErr)
	} else {
		progress := domain.JobProgress{
			ScrapedPlaces: totalResults,
		}
		if progressErr := h.jobs.UpdateProgress(r.Context(), id, progress); progressErr != nil {
			log.Printf("[SubmitResults] Job %s: WARNING - failed to update progress: %v", id, progressErr)
		} else {
			log.Printf("[SubmitResults] Job %s: Updated scraped_places to %d", id, totalResults)
		}
	}

	w.WriteHeader(http.StatusCreated)
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(jobs JobServiceInterface, results ResultServiceInterface) *JobHandler {
	return &JobHandler{
		jobs:    jobs,
		results: results,
	}
}

// NewJobHandlerWithCache creates a new JobHandler with caching support
func NewJobHandlerWithCache(jobs JobServiceInterface, results ResultServiceInterface, c cache.Cache) *JobHandler {
	return &JobHandler{
		jobs:    jobs,
		results: results,
		cache:   c,
	}
}

// invalidateJobCache invalidates all job-related cache entries
func (h *JobHandler) invalidateJobCache(ctx context.Context, jobID *uuid.UUID) {
	if h.cache == nil {
		return
	}

	// Invalidate job list cache
	if err := h.cache.DeleteByPattern(ctx, cache.KeyPrefixDashboardJobs+":*"); err != nil {
		log.Printf("[JobHandler] Warning: failed to invalidate job list cache: %v", err)
	}

	// Invalidate stats cache
	if err := h.cache.Delete(ctx, cache.KeyPrefixDashboardStats); err != nil {
		log.Printf("[JobHandler] Warning: failed to invalidate stats cache: %v", err)
	}

	// Invalidate specific job detail cache if jobID provided
	if jobID != nil {
		detailKey := fmt.Sprintf("%s:detail:%s", cache.KeyPrefixDashboardJobs, jobID.String())
		if err := h.cache.Delete(ctx, detailKey); err != nil {
			log.Printf("[JobHandler] Warning: failed to invalidate job detail cache: %v", err)
		}
	}
}

// CreateJobRequest represents the request body for creating a job
type CreateJobRequest struct {
	Name         string   `json:"name"`
	Keywords     []string `json:"keywords"`
	Lang         string   `json:"lang"`
	Lat          *float64 `json:"lat,omitempty"`
	Lon          *float64 `json:"lon,omitempty"`
	Zoom         int      `json:"zoom"`
	Radius       int      `json:"radius"`
	Depth        int      `json:"depth"`
	FastMode     bool     `json:"fast_mode"`
	ExtractEmail bool     `json:"extract_email"`
	MaxTime      int      `json:"max_time"` // seconds
	Proxies      []string `json:"proxies,omitempty"`
	Priority     int      `json:"priority"`

	// Geo coverage settings for area-wide scraping
	LocationName string              `json:"location_name,omitempty"`
	BoundingBox  *domain.BoundingBox `json:"boundingbox,omitempty"`
	CoverageMode domain.CoverageMode `json:"coverage_mode,omitempty"`
}

// Create handles POST /api/v2/jobs
func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("[JobHandler] Create request received")

	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	log.Printf("[JobHandler] Request decoded in %v: name=%s, keywords=%d", time.Since(start), req.Name, len(req.Keywords))

	// Validate required fields
	if req.Name == "" {
		RenderError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if len(req.Keywords) == 0 {
		RenderError(w, http.StatusBadRequest, "At least one keyword is required")
		return
	}

	// Set defaults
	if req.Lang == "" {
		req.Lang = "en"
	}
	if req.Zoom == 0 {
		req.Zoom = 15
	}
	if req.Radius == 0 {
		req.Radius = 10000
	}
	if req.Depth == 0 {
		req.Depth = 10
	}
	if req.MaxTime == 0 {
		req.MaxTime = 600 // 10 minutes default
	}

	// Convert to domain request
	domainReq := &domain.CreateJobRequest{
		Name:         req.Name,
		Keywords:     req.Keywords,
		Lang:         req.Lang,
		GeoLat:       req.Lat,
		GeoLon:       req.Lon,
		Zoom:         req.Zoom,
		Radius:       req.Radius,
		Depth:        req.Depth,
		FastMode:     req.FastMode,
		ExtractEmail: req.ExtractEmail,
		MaxTime:      req.MaxTime,
		Proxies:      req.Proxies,
		Priority:     req.Priority,
		// Geo coverage settings for area-wide scraping
		LocationName: req.LocationName,
		BoundingBox:  req.BoundingBox,
		CoverageMode: req.CoverageMode,
	}

	log.Printf("[JobHandler] Calling service.Create")
	serviceStart := time.Now()
	job, err := h.jobs.Create(r.Context(), domainReq)
	if err != nil {
		log.Printf("[JobHandler] Create FAILED after %v (service: %v): %v", time.Since(start), time.Since(serviceStart), err)
		RenderError(w, http.StatusInternalServerError, "Failed to create job")
		return
	}

	// Invalidate cache after successful create
	h.invalidateJobCache(r.Context(), &job.ID)

	log.Printf("[JobHandler] Create completed in %v (service: %v)", time.Since(start), time.Since(serviceStart))
	RenderJSON(w, http.StatusCreated, job)
}

// List handles GET /api/v2/jobs
func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Try cache if available
	if h.cache != nil {
		cacheKey := fmt.Sprintf("%s:list:page=%d:perPage=%d:status=%s",
			cache.KeyPrefixDashboardJobs, page, perPage, status)
		if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}
	}

	params := domain.JobListParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	// Parse status filter
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

	// Cache the response if cache available
	if h.cache != nil {
		cacheKey := fmt.Sprintf("%s:list:page=%d:perPage=%d:status=%s",
			cache.KeyPrefixDashboardJobs, page, perPage, status)
		if data, err := json.Marshal(response); err == nil {
			h.cache.Set(ctx, cacheKey, data, cache.TTLJobsList)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, response)
}

// GetByID handles GET /api/v2/jobs/{id}
func (h *JobHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

	// Try cache if available
	if h.cache != nil {
		cacheKey := fmt.Sprintf("%s:%s", cache.KeyPrefixDashboardJobs, id.String())
		if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}
	}

	job, err := h.jobs.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			RenderError(w, http.StatusNotFound, "Job not found")
		} else {
			RenderError(w, http.StatusInternalServerError, "Failed to retrieve job: "+err.Error())
		}
		return
	}

	// Cache the response if cache available
	if h.cache != nil {
		cacheKey := fmt.Sprintf("%s:%s", cache.KeyPrefixDashboardJobs, id.String())
		if data, err := json.Marshal(job); err == nil {
			h.cache.Set(ctx, cacheKey, data, cache.TTLJobDetail)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	RenderJSON(w, http.StatusOK, job)
}

// Delete handles DELETE /api/v2/jobs/{id}
func (h *JobHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if err := h.jobs.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			RenderError(w, http.StatusNotFound, "Job not found")
		} else {
			RenderError(w, http.StatusInternalServerError, "Failed to delete job: "+err.Error())
		}
		return
	}

	// Invalidate cache after successful delete
	h.invalidateJobCache(r.Context(), &id)

	w.WriteHeader(http.StatusNoContent)
}

// Pause handles POST /api/v2/jobs/{id}/pause
func (h *JobHandler) Pause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobs.Pause(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			RenderError(w, http.StatusNotFound, "Job not found")
		} else {
			RenderError(w, http.StatusBadRequest, "Failed to pause job: "+err.Error())
		}
		return
	}

	// Invalidate cache after successful pause
	h.invalidateJobCache(r.Context(), &id)

	RenderJSON(w, http.StatusOK, job)
}

// Resume handles POST /api/v2/jobs/{id}/resume
func (h *JobHandler) Resume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobs.Resume(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			RenderError(w, http.StatusNotFound, "Job not found")
		} else {
			RenderError(w, http.StatusBadRequest, "Failed to resume job: "+err.Error())
		}
		return
	}

	// Invalidate cache after successful resume
	h.invalidateJobCache(r.Context(), &id)

	RenderJSON(w, http.StatusOK, job)
}

// Cancel handles POST /api/v2/jobs/{id}/cancel
func (h *JobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobs.Cancel(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			RenderError(w, http.StatusNotFound, "Job not found")
		} else {
			RenderError(w, http.StatusBadRequest, "Failed to cancel job: "+err.Error())
		}
		return
	}

	// Invalidate cache after successful cancel
	h.invalidateJobCache(r.Context(), &id)

	RenderJSON(w, http.StatusOK, job)
}

// GetResults handles GET /api/v2/jobs/{id}/results
func (h *JobHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	results, total, err := h.results.ListByJobID(r.Context(), id, perPage, offset)
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
	RenderJSON(w, http.StatusOK, response)
}

// DownloadResults handles GET /api/v2/jobs/{id}/download
func (h *JobHandler) DownloadResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		h.downloadJSON(w, r, id)
	case "csv":
		h.downloadCSV(w, r, id)
	case "xlsx":
		h.downloadXLSX(w, r, id)
	default:
		RenderError(w, http.StatusBadRequest, "Invalid format. Use 'json', 'csv', or 'xlsx'")
	}
}

func (h *JobHandler) downloadJSON(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	// Create download context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), downloadTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=results-"+jobID.String()+".json")

	w.Write([]byte("["))
	first := true
	count := 0

	err := h.results.StreamByJobID(ctx, jobID, func(data []byte) error {
		if !first {
			w.Write([]byte(","))
		}
		first = false
		w.Write(data)
		count++

		// Flush every 100 records to prevent buffering timeout
		if count%100 == 0 {
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("error streaming JSON for job %s: %v", jobID, err)
	}

	w.Write([]byte("]"))
}

func (h *JobHandler) downloadCSV(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	// Create download context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), downloadTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=results-"+jobID.String()+".csv")

	availableColumns := getAvailableColumns()
	selectedColumns := parseSelectedColumns(r.URL.Query().Get("columns"), availableColumns)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write Header
	if err := writer.Write(selectedColumns); err != nil {
		return
	}

	count := 0
	err := h.results.StreamByJobID(ctx, jobID, func(data []byte) error {
		var entry gmaps.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		record := make([]string, len(selectedColumns))
		for i, col := range selectedColumns {
			record[i] = availableColumns[col](&entry)
		}

		if err := writer.Write(record); err != nil {
			return err
		}

		count++
		// Flush every 100 records to prevent buffering timeout
		if count%100 == 0 {
			writer.Flush()
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("error streaming CSV for job %s: %v", jobID, err)
	}
}

func (h *JobHandler) downloadXLSX(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	// Create download context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), downloadTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=results-"+jobID.String()+".xlsx")

	availableColumns := getAvailableColumns()

	// Parse requested columns
	selectedColumns := parseSelectedColumns(r.URL.Query().Get("columns"), availableColumns)

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	// Write header row
	for i, col := range selectedColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col)
	}

	// Style the header
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	lastCol, _ := excelize.CoordinatesToCellName(len(selectedColumns), 1)
	f.SetCellStyle(sheetName, "A1", lastCol, headerStyle)

	rowNum := 2
	err := h.results.StreamByJobID(ctx, jobID, func(data []byte) error {
		var entry gmaps.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		for i, col := range selectedColumns {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowNum)
			f.SetCellValue(sheetName, cell, availableColumns[col](&entry))
		}
		rowNum++
		return nil
	})

	if err != nil {
		log.Printf("error streaming results for XLSX job %s: %v", jobID, err)
	}

	// Auto-fit column widths (approximate)
	for i := range selectedColumns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, colName, colName, 15)
	}

	// Write to response
	if err := f.Write(w); err != nil {
		log.Printf("error writing XLSX to response: %v", err)
	}
}

// getAvailableColumns returns the map of available export columns
func getAvailableColumns() map[string]func(e *gmaps.Entry) string {
	return map[string]func(e *gmaps.Entry) string{
		"Title":           func(e *gmaps.Entry) string { return e.Title },
		"Address":         func(e *gmaps.Entry) string { return e.Address },
		"Phone":           func(e *gmaps.Entry) string { return e.Phone },
		"Website":         func(e *gmaps.Entry) string { return e.WebSite },
		"Category":        func(e *gmaps.Entry) string { return e.Category },
		"Rating":          func(e *gmaps.Entry) string { return fmt.Sprintf("%.1f", e.ReviewRating) },
		"Reviews":         func(e *gmaps.Entry) string { return fmt.Sprintf("%d", e.ReviewCount) },
		"Latitude":        func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Latitude) },
		"Longitude":       func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Longitude) },
		"Place ID":        func(e *gmaps.Entry) string { return e.PlaceID },
		"Google Maps URL": func(e *gmaps.Entry) string { return e.Link },
		"Description":     func(e *gmaps.Entry) string { return e.Description },
		"Status":          func(e *gmaps.Entry) string { return e.Status },
		"Timezone":        func(e *gmaps.Entry) string { return e.Timezone },
		"Price Range":     func(e *gmaps.Entry) string { return e.PriceRange },
		"Data ID":         func(e *gmaps.Entry) string { return e.DataID },
		"Email": func(e *gmaps.Entry) string { return strings.Join(e.Emails, ", ") },
		"Opening Hours": func(e *gmaps.Entry) string {
			var parts []string
			for day, hours := range e.OpenHours {
				parts = append(parts, fmt.Sprintf("%s: %s", day, strings.Join(hours, ", ")))
			}
			return strings.Join(parts, "; ")
		},
	}
}

// parseSelectedColumns parses and validates requested columns
func parseSelectedColumns(colsParam string, availableColumns map[string]func(e *gmaps.Entry) string) []string {
	var selectedColumns []string
	if colsParam != "" {
		requested := strings.Split(colsParam, ",")
		for _, col := range requested {
			col = strings.TrimSpace(col)
			if _, ok := availableColumns[col]; ok {
				selectedColumns = append(selectedColumns, col)
			}
		}
	}

	// Default columns if none selected or invalid
	if len(selectedColumns) == 0 {
		selectedColumns = []string{
			"Title", "Address", "Phone", "Website", "Category", "Rating", "Reviews",
			"Latitude", "Longitude", "Place ID", "Google Maps URL",
		}
	}
	return selectedColumns
}

// GetStats handles GET /api/v2/jobs/stats
func (h *JobHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats, err := h.jobs.GetStats(r.Context())
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusOK, stats)
}

func parseJobID(r *http.Request) (uuid.UUID, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	return uuid.Parse(idStr)
}
