package api

import (
	"net/http"

	"github.com/sadewadee/google-scraper/internal/api/handlers"
)

// Router sets up all API routes
type Router struct {
	mux     *http.ServeMux
	jobs    *handlers.JobHandler
	workers *handlers.WorkerHandler
	stats   *handlers.StatsHandler
	proxy   *handlers.ProxyHandler
	results *handlers.ResultHandler
}

// NewRouter creates a new Router
func NewRouter(
	jobs *handlers.JobHandler,
	workers *handlers.WorkerHandler,
	stats *handlers.StatsHandler,
	proxy *handlers.ProxyHandler,
	results *handlers.ResultHandler,
) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		jobs:    jobs,
		workers: workers,
		stats:   stats,
		proxy:   proxy,
		results: results,
	}
}

// Setup configures all routes
func (r *Router) Setup(token string) http.Handler {
	// Health check endpoint (no auth required)
	r.mux.HandleFunc("/health", r.healthCheck)
	r.mux.HandleFunc("/api/v2/health", r.healthCheck)

	// Stats endpoint
	r.mux.HandleFunc("/api/v2/stats", r.stats.GetDashboardStats)

	// ProxyGate endpoints
	r.mux.HandleFunc("/api/v2/proxygate/stats", r.proxy.GetStats)
	r.mux.HandleFunc("/api/v2/proxygate/sources", r.handleProxySources)
	r.mux.HandleFunc("/api/v2/proxygate/sources/{id}", r.handleProxySource)
	r.mux.HandleFunc("/api/v2/proxygate/refresh", r.proxy.Refresh)

	// Job endpoints
	r.mux.HandleFunc("/api/v2/jobs", r.handleJobs)
	r.mux.HandleFunc("/api/v2/jobs/stats", r.jobs.GetStats)
	r.mux.HandleFunc("/api/v2/jobs/{id}", r.handleJob)
	r.mux.HandleFunc("/api/v2/jobs/{id}/pause", r.jobs.Pause)
	r.mux.HandleFunc("/api/v2/jobs/{id}/resume", r.jobs.Resume)
	r.mux.HandleFunc("/api/v2/jobs/{id}/cancel", r.jobs.Cancel)
	r.mux.HandleFunc("/api/v2/jobs/{id}/results", r.handleJobResults)
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

	// Global results endpoints (database view)
	r.mux.HandleFunc("/api/v2/results", r.results.List)
	r.mux.HandleFunc("/api/v2/results/download", r.results.Download)

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

// handleProxySources routes requests for /api/v2/proxygate/sources
func (r *Router) handleProxySources(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.proxy.GetSources(w, req)
	case http.MethodPost:
		r.proxy.AddSource(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProxySource routes requests for /api/v2/proxygate/sources/{id}
func (r *Router) handleProxySource(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodDelete:
		r.proxy.DeleteSource(w, req)
	case http.MethodPatch:
		r.proxy.UpdateSource(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// healthCheck returns a simple health status (no auth required)
func (r *Router) healthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleJobResults routes requests for /api/v2/jobs/{id}/results
func (r *Router) handleJobResults(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.jobs.GetResults(w, req)
	case http.MethodPost:
		r.jobs.SubmitResults(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
