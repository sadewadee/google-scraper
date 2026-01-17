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

	// Business listings handler (normalized data from business_listings table)
	businessListings *handlers.BusinessListingHandler

	// Cached handlers for read operations (optional, set via SetCachedHandlers)
	cachedJobs    *handlers.CachedJobHandler
	cachedStats   *handlers.CachedStatsHandler
	cachedResults *handlers.CachedResultHandler
}

// NewRouter creates a new Router
func NewRouter(
	jobs *handlers.JobHandler,
	workers *handlers.WorkerHandler,
	stats *handlers.StatsHandler,
	proxy *handlers.ProxyHandler,
	results *handlers.ResultHandler,
	businessListings *handlers.BusinessListingHandler,
) *Router {
	return &Router{
		mux:              http.NewServeMux(),
		jobs:             jobs,
		workers:          workers,
		stats:            stats,
		proxy:            proxy,
		results:          results,
		businessListings: businessListings,
	}
}

// SetCachedHandlers sets optional cached handlers for read operations
func (r *Router) SetCachedHandlers(
	cachedJobs *handlers.CachedJobHandler,
	cachedStats *handlers.CachedStatsHandler,
	cachedResults *handlers.CachedResultHandler,
) {
	r.cachedJobs = cachedJobs
	r.cachedStats = cachedStats
	r.cachedResults = cachedResults
}

// Setup configures all routes
func (r *Router) Setup(token string) http.Handler {
	// Health check endpoint (no auth required)
	r.mux.HandleFunc("/health", r.healthCheck)
	r.mux.HandleFunc("/api/v2/health", r.healthCheck)

	// Stats endpoint - use cached handler if available
	if r.cachedStats != nil {
		r.mux.HandleFunc("/api/v2/stats", r.cachedStats.GetDashboardStats)
	} else {
		r.mux.HandleFunc("/api/v2/stats", r.stats.GetDashboardStats)
	}

	// ProxyGate endpoints
	r.mux.HandleFunc("/api/v2/proxygate/stats", r.proxy.GetProxyStats)
	r.mux.HandleFunc("/api/v2/proxygate/sources", r.handleProxySources)
	r.mux.HandleFunc("/api/v2/proxygate/sources/{id}", r.handleProxySource)
	r.mux.HandleFunc("/api/v2/proxygate/refresh", r.proxy.Refresh)
	r.mux.HandleFunc("/api/v2/proxygate/proxies", r.handleProxies)
	r.mux.HandleFunc("/api/v2/proxygate/proxies/bulk", r.proxy.AddProxiesBulk)
	r.mux.HandleFunc("/api/v2/proxygate/proxies/cleanup", r.proxy.DeleteDeadProxies)
	r.mux.HandleFunc("/api/v2/proxygate/proxies/{id}", r.handleProxy)

	// Job endpoints
	r.mux.HandleFunc("/api/v2/jobs", r.handleJobs)
	r.mux.HandleFunc("/api/v2/jobs/stats", r.handleJobStats)
	r.mux.HandleFunc("/api/v2/jobs/{id}", r.handleJob)
	r.mux.HandleFunc("/api/v2/jobs/{id}/pause", r.jobs.Pause)
	r.mux.HandleFunc("/api/v2/jobs/{id}/resume", r.jobs.Resume)
	r.mux.HandleFunc("/api/v2/jobs/{id}/cancel", r.jobs.Cancel)
	r.mux.HandleFunc("/api/v2/jobs/{id}/results", r.handleJobResults)
	r.mux.HandleFunc("/api/v2/jobs/{id}/download", r.handleJobDownload)

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

	// Global results endpoints - use business_listings table via BusinessListingHandler
	// (Normalized data with proper columns, filtering, and export formats)
	if r.businessListings != nil {
		r.mux.HandleFunc("/api/v2/results", r.businessListings.List)
		r.mux.HandleFunc("/api/v2/results/download", r.businessListings.Download)
		r.mux.HandleFunc("/api/v2/results/categories", r.businessListings.GetCategories)
		r.mux.HandleFunc("/api/v2/results/cities", r.businessListings.GetCities)
		r.mux.HandleFunc("/api/v2/results/stats", r.businessListings.GetStats)
		r.mux.HandleFunc("/api/v2/results/columns", r.businessListings.GetAvailableColumns)
	} else if r.cachedResults != nil {
		// Fallback to cached raw results handler
		r.mux.HandleFunc("/api/v2/results", r.cachedResults.List)
		r.mux.HandleFunc("/api/v2/results/download", r.results.Download)
	} else {
		// Fallback to raw results handler
		r.mux.HandleFunc("/api/v2/results", r.results.List)
		r.mux.HandleFunc("/api/v2/results/download", r.results.Download)
	}

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
		// Use cached handler for read operations if available
		if r.cachedJobs != nil {
			r.cachedJobs.List(w, req)
		} else {
			r.jobs.List(w, req)
		}
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
		// Use cached handler for read operations if available
		if r.cachedJobs != nil {
			r.cachedJobs.GetByID(w, req)
		} else {
			r.jobs.GetByID(w, req)
		}
	case http.MethodDelete:
		r.jobs.Delete(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleJobStats routes requests for /api/v2/jobs/stats
func (r *Router) handleJobStats(w http.ResponseWriter, req *http.Request) {
	// Use cached handler if available
	if r.cachedJobs != nil {
		r.cachedJobs.GetStats(w, req)
	} else {
		r.jobs.GetStats(w, req)
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

// handleProxies routes requests for /api/v2/proxygate/proxies
func (r *Router) handleProxies(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.proxy.ListProxies(w, req)
	case http.MethodPost:
		r.proxy.AddProxy(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProxy routes requests for /api/v2/proxygate/proxies/{id}
func (r *Router) handleProxy(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPatch:
		r.proxy.UpdateProxyStatus(w, req)
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
		// Use business listings handler (normalized data from business_listings table)
		if r.businessListings != nil {
			r.businessListings.ListByJobID(w, req)
		} else if r.cachedJobs != nil {
			// Fallback to cached raw results
			r.cachedJobs.GetResults(w, req)
		} else {
			r.jobs.GetResults(w, req)
		}
	case http.MethodPost:
		r.jobs.SubmitResults(w, req)
	default:
		handlers.RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleJobDownload routes requests for /api/v2/jobs/{id}/download
func (r *Router) handleJobDownload(w http.ResponseWriter, req *http.Request) {
	// Use business listings handler (normalized data from business_listings table)
	if r.businessListings != nil {
		r.businessListings.DownloadByJobID(w, req)
	} else {
		// Fallback to raw results handler
		r.jobs.DownloadResults(w, req)
	}
}
