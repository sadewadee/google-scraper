package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/internal/domain"
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
	ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error)
	StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error
}

// JobHandler handles job-related HTTP requests
type JobHandler struct {
	jobs    JobServiceInterface
	results ResultServiceInterface
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(jobs JobServiceInterface, results ResultServiceInterface) *JobHandler {
	return &JobHandler{
		jobs:    jobs,
		results: results,
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
}

// Create handles POST /api/v2/jobs
func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

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
	}

	job, err := h.jobs.Create(r.Context(), domainReq)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to create job: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusCreated, job)
}

// List handles GET /api/v2/jobs
func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	params := domain.JobListParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.JobStatus(status)
		params.Status = &s
	}

	jobs, total, err := h.jobs.List(r.Context(), params)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to list jobs: "+err.Error())
		return
	}

	response := NewPaginatedResponse(jobs, total, page, perPage)
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

	job, err := h.jobs.GetByID(r.Context(), id)
	if err != nil {
		RenderError(w, http.StatusNotFound, "Job not found")
		return
	}

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
		RenderError(w, http.StatusInternalServerError, "Failed to delete job: "+err.Error())
		return
	}

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
		RenderError(w, http.StatusBadRequest, err.Error())
		return
	}

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
		RenderError(w, http.StatusBadRequest, err.Error())
		return
	}

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
		RenderError(w, http.StatusBadRequest, err.Error())
		return
	}

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
	default:
		RenderError(w, http.StatusBadRequest, "Invalid format. Use 'json' or 'csv'")
	}
}

func (h *JobHandler) downloadJSON(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=results-"+jobID.String()+".json")

	w.Write([]byte("["))
	first := true

	err := h.results.StreamByJobID(r.Context(), jobID, func(data []byte) error {
		if !first {
			w.Write([]byte(","))
		}
		first = false
		w.Write(data)
		return nil
	})

	if err != nil {
		// Already started writing, can't change status
		w.Write([]byte("]"))
		return
	}

	w.Write([]byte("]"))
}

func (h *JobHandler) downloadCSV(w http.ResponseWriter, r *http.Request, jobID uuid.UUID) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=results-"+jobID.String()+".csv")

	// Define available columns
	availableColumns := map[string]func(e *gmaps.Entry) string{
		"Title":           func(e *gmaps.Entry) string { return e.Title },
		"Address":         func(e *gmaps.Entry) string { return e.Address },
		"Phone":           func(e *gmaps.Entry) string { return e.Phone },
		"Website":         func(e *gmaps.Entry) string { return e.WebSite },
		"Category":        func(e *gmaps.Entry) string { return e.Category },
		"Rating":          func(e *gmaps.Entry) string { return fmt.Sprintf("%.1f", e.ReviewRating) },
		"Reviews":         func(e *gmaps.Entry) string { return fmt.Sprintf("%d", e.ReviewCount) },
		"Latitude":        func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Latitude) },
		"Longitude":       func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Longtitude) },
		"Place ID":        func(e *gmaps.Entry) string { return e.PlaceID },
		"Google Maps URL": func(e *gmaps.Entry) string { return e.Link },
		"Description":     func(e *gmaps.Entry) string { return e.Description },
		"Status":          func(e *gmaps.Entry) string { return e.Status },
		"Timezone":        func(e *gmaps.Entry) string { return e.Timezone },
		"Price Range":     func(e *gmaps.Entry) string { return e.PriceRange },
		"Data ID":         func(e *gmaps.Entry) string { return e.DataID },
	}

	// Parse requested columns
	var selectedColumns []string
	colsParam := r.URL.Query().Get("columns")
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

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write Header
	if err := writer.Write(selectedColumns); err != nil {
		return
	}

	err := h.results.StreamByJobID(r.Context(), jobID, func(data []byte) error {
		var entry gmaps.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		record := make([]string, len(selectedColumns))
		for i, col := range selectedColumns {
			record[i] = availableColumns[col](&entry)
		}

		return writer.Write(record)
	})

	if err != nil {
		// Log error if needed
	}
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
