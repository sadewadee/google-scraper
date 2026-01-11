package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// WorkerServiceInterface defines the worker service methods
type WorkerServiceInterface interface {
	Register(ctx context.Context, workerID string) (*domain.Worker, error)
	Heartbeat(ctx context.Context, hb *domain.WorkerHeartbeat) error
	List(ctx context.Context, params domain.WorkerListParams) ([]*domain.Worker, error)
	GetByID(ctx context.Context, id string) (*domain.Worker, error)
	GetStats(ctx context.Context) (*domain.WorkerStats, error)
	ClaimJob(ctx context.Context, workerID string) (*domain.Job, error)
	ReleaseJob(ctx context.Context, jobID uuid.UUID, workerID string) error
	CompleteJob(ctx context.Context, jobID uuid.UUID, workerID string, placesScraped int) error
	FailJob(ctx context.Context, jobID uuid.UUID, workerID string, errMsg string) error
	Unregister(ctx context.Context, workerID string) error
}

// WorkerHandler handles worker-related HTTP requests
type WorkerHandler struct {
	workers WorkerServiceInterface
}

// NewWorkerHandler creates a new WorkerHandler
func NewWorkerHandler(workers WorkerServiceInterface) *WorkerHandler {
	return &WorkerHandler{
		workers: workers,
	}
}

// RegisterRequest represents the request body for worker registration
type RegisterRequest struct {
	WorkerID string `json:"worker_id"`
}

// HeartbeatRequest represents the request body for worker heartbeat
type HeartbeatRequest struct {
	WorkerID     string              `json:"worker_id"`
	Hostname     string              `json:"hostname,omitempty"`
	Status       domain.WorkerStatus `json:"status"`
	CurrentJobID *uuid.UUID          `json:"current_job_id,omitempty"`
}

// CompleteJobRequest represents the request body for completing a job
type CompleteJobRequest struct {
	JobID         uuid.UUID `json:"job_id"`
	PlacesScraped int       `json:"places_scraped"`
}

// FailJobRequest represents the request body for failing a job
type FailJobRequest struct {
	JobID   uuid.UUID `json:"job_id"`
	Message string    `json:"message"`
}

// ReleaseJobRequest represents the request body for releasing a job
type ReleaseJobRequest struct {
	JobID uuid.UUID `json:"job_id"`
}

// Register handles POST /api/v2/workers/register
func (h *WorkerHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.WorkerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	worker, err := h.workers.Register(r.Context(), req.WorkerID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to register worker: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusCreated, worker)
}

// Heartbeat handles POST /api/v2/workers/heartbeat
func (h *WorkerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.WorkerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	hb := &domain.WorkerHeartbeat{
		WorkerID:     req.WorkerID,
		Hostname:     req.Hostname,
		Status:       req.Status,
		CurrentJobID: req.CurrentJobID,
	}

	if err := h.workers.Heartbeat(r.Context(), hb); err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to update heartbeat: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ClaimJob handles POST /api/v2/workers/{id}/claim
func (h *WorkerHandler) ClaimJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	job, err := h.workers.ClaimJob(r.Context(), workerID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to claim job: "+err.Error())
		return
	}

	if job == nil {
		// No pending jobs
		RenderJSON(w, http.StatusOK, map[string]interface{}{
			"job": nil,
		})
		return
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"job": job,
	})
}

// CompleteJob handles POST /api/v2/workers/{id}/complete
func (h *WorkerHandler) CompleteJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	var req CompleteJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.workers.CompleteJob(r.Context(), req.JobID, workerID, req.PlacesScraped); err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to complete job: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// FailJob handles POST /api/v2/workers/{id}/fail
func (h *WorkerHandler) FailJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	var req FailJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.workers.FailJob(r.Context(), req.JobID, workerID, req.Message); err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to fail job: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReleaseJob handles POST /api/v2/workers/{id}/release
func (h *WorkerHandler) ReleaseJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	var req ReleaseJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.workers.ReleaseJob(r.Context(), req.JobID, workerID); err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to release job: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v2/workers
func (h *WorkerHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	params := domain.WorkerListParams{}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.WorkerStatus(status)
		params.Status = &s
	}

	workers, err := h.workers.List(r.Context(), params)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to list workers: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusOK, workers)
}

// GetByID handles GET /api/v2/workers/{id}
func (h *WorkerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	worker, err := h.workers.GetByID(r.Context(), workerID)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get worker: "+err.Error())
		return
	}

	if worker == nil {
		RenderError(w, http.StatusNotFound, "Worker not found")
		return
	}

	RenderJSON(w, http.StatusOK, worker)
}

// GetStats handles GET /api/v2/workers/stats
func (h *WorkerHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats, err := h.workers.GetStats(r.Context())
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusOK, stats)
}

// Unregister handles DELETE /api/v2/workers/{id}
func (h *WorkerHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	workerID := r.PathValue("id")
	if workerID == "" {
		RenderError(w, http.StatusBadRequest, "Worker ID is required")
		return
	}

	if err := h.workers.Unregister(r.Context(), workerID); err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to unregister worker: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
