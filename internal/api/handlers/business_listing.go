package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/service"
)

// BusinessListingHandler handles business listing endpoints
type BusinessListingHandler struct {
	svc *service.BusinessListingService
}

// NewBusinessListingHandler creates a new handler
func NewBusinessListingHandler(svc *service.BusinessListingService) *BusinessListingHandler {
	return &BusinessListingHandler{svc: svc}
}

// List handles GET /api/v2/results (global business listings)
func (h *BusinessListingHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filter := domain.BusinessListingFilter{
		Page:      1,
		PerPage:   25,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if page, err := strconv.Atoi(p); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.Atoi(l); err == nil && limit > 0 && limit <= 100 {
			filter.PerPage = limit
		}
	}

	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = s
	}

	if c := r.URL.Query().Get("category"); c != "" {
		filter.Category = c
	}

	if city := r.URL.Query().Get("city"); city != "" {
		filter.City = city
	}

	if country := r.URL.Query().Get("country"); country != "" {
		filter.Country = country
	}

	if rating := r.URL.Query().Get("min_rating"); rating != "" {
		if r, err := strconv.ParseFloat(rating, 64); err == nil {
			filter.MinRating = &r
		}
	}

	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	// Parse email filters
	if hasEmail := r.URL.Query().Get("has_email"); hasEmail != "" {
		val := strings.ToLower(hasEmail) == "true" || hasEmail == "1"
		filter.HasEmail = &val
	}

	if emailStatus := r.URL.Query().Get("email_status"); emailStatus != "" {
		filter.EmailStatus = strings.ToLower(emailStatus)
	}

	listings, total, err := h.svc.List(ctx, filter)
	if err != nil {
		log.Printf("[BusinessListingHandler] List error: %v", err)
		h.jsonError(w, "Failed to fetch listings", http.StatusInternalServerError)
		return
	}

	totalPages := (total + filter.PerPage - 1) / filter.PerPage

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": listings,
		"meta": map[string]interface{}{
			"page":        filter.Page,
			"per_page":    filter.PerPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// Download handles GET /api/v2/results/download (export global listings)
func (h *BusinessListingHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Parse columns
	var columns []string
	if cols := r.URL.Query().Get("columns"); cols != "" {
		columns = strings.Split(cols, ",")
		// Validate columns
		validCols := make(map[string]bool)
		for _, c := range h.svc.AvailableColumns() {
			validCols[c] = true
		}
		for _, c := range columns {
			if !validCols[c] {
				h.jsonError(w, "Invalid column: "+c, http.StatusBadRequest)
				return
			}
		}
	}

	// Parse filter
	filter := domain.BusinessListingFilter{}

	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = s
	}

	if c := r.URL.Query().Get("category"); c != "" {
		filter.Category = c
	}

	if city := r.URL.Query().Get("city"); city != "" {
		filter.City = city
	}

	if country := r.URL.Query().Get("country"); country != "" {
		filter.Country = country
	}

	if rating := r.URL.Query().Get("min_rating"); rating != "" {
		if r, err := strconv.ParseFloat(rating, 64); err == nil {
			filter.MinRating = &r
		}
	}

	// Parse email filters
	if hasEmail := r.URL.Query().Get("has_email"); hasEmail != "" {
		val := strings.ToLower(hasEmail) == "true" || hasEmail == "1"
		filter.HasEmail = &val
	}

	if emailStatus := r.URL.Query().Get("email_status"); emailStatus != "" {
		filter.EmailStatus = strings.ToLower(emailStatus)
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=business_listings.csv")
		if err := h.svc.ExportCSV(ctx, w, filter, columns); err != nil {
			log.Printf("[BusinessListingHandler] ExportCSV error: %v", err)
			return
		}
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=business_listings.json")
		if err := h.svc.ExportJSON(ctx, w, filter); err != nil {
			log.Printf("[BusinessListingHandler] ExportJSON error: %v", err)
			return
		}
	case "xlsx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=business_listings.xlsx")
		if err := h.svc.ExportXLSX(ctx, w, filter, columns); err != nil {
			log.Printf("[BusinessListingHandler] ExportXLSX error: %v", err)
			return
		}
	default:
		h.jsonError(w, "Invalid format. Supported: csv, json, xlsx", http.StatusBadRequest)
	}
}

// ListByJobID handles GET /api/v2/jobs/{id}/results
func (h *BusinessListingHandler) ListByJobID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract job ID from path
	jobID := extractJobIDFromPath(r.URL.Path)
	if jobID == "" {
		h.jsonError(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(jobID); err != nil {
		h.jsonError(w, "Invalid job ID format", http.StatusBadRequest)
		return
	}

	// Parse pagination
	page := 1
	limit := 25

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	offset := (page - 1) * limit

	listings, total, err := h.svc.ListByJobID(ctx, jobID, limit, offset)
	if err != nil {
		log.Printf("[BusinessListingHandler] ListByJobID error: %v", err)
		h.jsonError(w, "Failed to fetch listings", http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": listings,
		"meta": map[string]interface{}{
			"page":        page,
			"per_page":    limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// DownloadByJobID handles GET /api/v2/jobs/{id}/download
func (h *BusinessListingHandler) DownloadByJobID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract job ID from path
	jobID := extractJobIDFromPath(r.URL.Path)
	if jobID == "" {
		h.jsonError(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(jobID); err != nil {
		h.jsonError(w, "Invalid job ID format", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Parse columns
	var columns []string
	if cols := r.URL.Query().Get("columns"); cols != "" {
		columns = strings.Split(cols, ",")
		// Validate columns
		validCols := make(map[string]bool)
		for _, c := range h.svc.AvailableColumns() {
			validCols[c] = true
		}
		for _, c := range columns {
			if !validCols[c] {
				h.jsonError(w, "Invalid column: "+c, http.StatusBadRequest)
				return
			}
		}
	}

	filename := "job_" + jobID[:8]

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".csv")
		if err := h.svc.ExportCSVByJobID(ctx, w, jobID, columns); err != nil {
			log.Printf("[BusinessListingHandler] ExportCSVByJobID error: %v", err)
			return
		}
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".json")
		if err := h.svc.ExportJSONByJobID(ctx, w, jobID); err != nil {
			log.Printf("[BusinessListingHandler] ExportJSONByJobID error: %v", err)
			return
		}
	case "xlsx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".xlsx")
		if err := h.svc.ExportXLSXByJobID(ctx, w, jobID, columns); err != nil {
			log.Printf("[BusinessListingHandler] ExportXLSXByJobID error: %v", err)
			return
		}
	default:
		h.jsonError(w, "Invalid format. Supported: csv, json, xlsx", http.StatusBadRequest)
	}
}

// GetCategories handles GET /api/v2/results/categories
func (h *BusinessListingHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	categories, err := h.svc.GetCategories(ctx, limit)
	if err != nil {
		log.Printf("[BusinessListingHandler] GetCategories error: %v", err)
		h.jsonError(w, "Failed to fetch categories", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": categories,
	})
}

// GetCities handles GET /api/v2/results/cities
func (h *BusinessListingHandler) GetCities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	cities, err := h.svc.GetCities(ctx, limit)
	if err != nil {
		log.Printf("[BusinessListingHandler] GetCities error: %v", err)
		h.jsonError(w, "Failed to fetch cities", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": cities,
	})
}

// GetStats handles GET /api/v2/results/stats
func (h *BusinessListingHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.svc.Stats(ctx)
	if err != nil {
		log.Printf("[BusinessListingHandler] GetStats error: %v", err)
		h.jsonError(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": stats,
	})
}

// GetAvailableColumns handles GET /api/v2/results/columns
func (h *BusinessListingHandler) GetAvailableColumns(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data": h.svc.AvailableColumns(),
	})
}

// extractJobIDFromPath extracts job ID from paths like /api/v2/jobs/{id}/results
func extractJobIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "jobs" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// jsonResponse sends a JSON response
func (h *BusinessListingHandler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// jsonError sends a JSON error response
func (h *BusinessListingHandler) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}
