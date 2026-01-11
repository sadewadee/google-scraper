package api

import (
	"net/http"

	"github.com/gosom/google-maps-scraper/internal/api/handlers"
)

// Router sets up all API routes
type Router struct {
	mux     *http.ServeMux
	jobs    *handlers.JobHandler
	workers *handlers.WorkerHandler
	stats   *handlers.StatsHandler
}

// NewRouter creates a new Router
func NewRouter(
	jobs *handlers.JobHandler,
	workers *handlers.WorkerHandler,
	stats *handlers.StatsHandler,
) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		jobs:    jobs,
		workers: workers,
		stats:   stats,
	}
}

// Setup configures all routes
func (r *Router) Setup(token string) http.Handler {
	// Stats endpoint
	r.mux.HandleFunc("/api/v2/stats", r.stats.GetDashboardStats)

	// Job endpoints
	r.mux.HandleFunc("/api/v2/jobs", r.handleJobs)
	r.mux.HandleFunc("/api/v2/jobs/stats", r.jobs.GetStats)
	r.mux.HandleFunc("/api/v2/jobs/{id}", r.handleJob)
	r.mux.HandleFunc("/api/v2/jobs/{id}/pause", r.jobs.Pause)
	r.mux.HandleFunc("/api/v2/jobs/{id}/resume", r.jobs.Resume)
	r.mux.HandleFunc("/api/v2/jobs/{id}/cancel", r.jobs.Cancel)
	r.mux.HandleFunc("/api/v2/jobs/{id}/results", r.jobs.GetResults)
	r.mux.HandleFunc("/api/v2/jobs/{id}/download", r.jobs.DownloadResults)

	// Worker endpoints
	r.mux.HandleFunc("/api/v2/workers", r.workers.List)
	r.mux.HandleFunc("/api/v2/workers/register", r.workers.Register)
	r.mux.HandleFunc("/api/v2/workers/heartbeat", r.workers.Heartbeat)
	r.mux.HandleFunc("/api/v2/workers/stats", r.workers.GetStats)
	r.mux.HandleFunc("/api/v2/workers/{id}", r.handleWorker)
	r.mux.HandleFunc("/api/v2/workers/{id}/claim", r.workers.ClaimJob)
	r.mux.HandleFunc("/api/v2/workers/{id}/complete", r.workers.CompleteJob)
	r.mux.HandleFunc("/api/v2/workers/{id}/fail", r.workers.FailJob)
	r.mux.HandleFunc("/api/v2/workers/{id}/release", r.workers.ReleaseJob)

	// Apply middleware
	return Chain(r.mux,
		Recovery,
		Logger,
		CORS,
		SecurityHeaders,
		Auth(token),
	)
}

// handleJobs routes requests for /api/v2/jobs
func (r *Router) handleJobs(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.jobs.List(w, req)
	case http.MethodPost:
		r.jobs.Create(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleJob routes requests for /api/v2/jobs/{id}
func (r *Router) handleJob(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.jobs.GetByID(w, req)
	case http.MethodDelete:
		r.jobs.Delete(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleWorker routes requests for /api/v2/workers/{id}
func (r *Router) handleWorker(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.workers.GetByID(w, req)
	case http.MethodDelete:
		r.workers.Unregister(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
