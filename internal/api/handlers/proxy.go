package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/proxygate"
)

type ProxyHandler struct {
	pg           *proxygate.ProxyGate
	repo         domain.ProxyRepository
	proxyListRepo domain.ProxyListRepository
}

func NewProxyHandler(pg *proxygate.ProxyGate, repo domain.ProxyRepository) *ProxyHandler {
	return &ProxyHandler{pg: pg, repo: repo}
}

// SetProxyListRepo sets the proxy list repository for listing individual proxies
func (h *ProxyHandler) SetProxyListRepo(repo domain.ProxyListRepository) {
	h.proxyListRepo = repo
}

func (h *ProxyHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		// Return empty stats if not enabled
		RenderJSON(w, http.StatusOK, map[string]interface{}{
			"data": map[string]interface{}{
				"total_proxies":   0,
				"healthy_proxies": 0,
				"last_updated":    "not enabled",
			},
		})
		return
	}

	total, healthy, lastUpdated := h.pg.GetStats()

	lastUpdatedStr := "never"
	if !lastUpdated.IsZero() {
		lastUpdatedStr = lastUpdated.Format(time.RFC3339)
	}

	stats := map[string]interface{}{
		"total_proxies":   total,
		"healthy_proxies": healthy,
		"last_updated":    lastUpdatedStr,
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"data": stats,
	})
}

func (h *ProxyHandler) GetSources(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		// Return empty list
		RenderJSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
		})
		return
	}

	if h.repo != nil {
		sources, err := h.repo.List(r.Context())
		if err != nil {
			log.Printf("Failed to list proxy sources: %v", err)
			RenderError(w, http.StatusInternalServerError, "Failed to list proxy sources")
			return
		}

		var response []map[string]interface{}
		for _, s := range sources {
			response = append(response, map[string]interface{}{
				"id":         s.ID,
				"url":        s.URL,
				"active":     true,
				"status":     "ok",
				"created_at": s.CreatedAt,
			})
		}
		RenderJSON(w, http.StatusOK, map[string]interface{}{
			"data": response,
		})
		return
	}

	sources := h.pg.GetSources()
	var response []map[string]interface{}

	for i, s := range sources {
		response = append(response, map[string]interface{}{
			"id":     i + 1,
			"url":    s,
			"active": true,
			"status": "ok",
		})
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"data": response,
	})
}

func (h *ProxyHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		RenderJSON(w, http.StatusOK, map[string]string{"message": "ProxyGate disabled, ignoring refresh"})
		return
	}

	if err := h.pg.Refresh(r.Context()); err != nil {
		log.Printf("Failed to refresh proxies: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to refresh proxies")
		return
	}

	RenderJSON(w, http.StatusOK, map[string]string{"message": "Refresh triggered"})
}

// Stubs for other methods to satisfy frontend
func (h *ProxyHandler) AddSource(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		RenderError(w, http.StatusServiceUnavailable, "ProxyGate disabled")
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		RenderError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Validate URL format
	if _, err := url.ParseRequestURI(req.URL); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid URL format")
		return
	}

	// Persist
	var id int64
	var createdAt time.Time

	if h.repo != nil {
		source, err := h.repo.Create(r.Context(), req.URL)
		if err != nil {
			log.Printf("Failed to create proxy source: %v", err)
			RenderError(w, http.StatusInternalServerError, "Failed to create proxy source")
			return
		}
		id = source.ID
		createdAt = source.CreatedAt
	} else {
		id = time.Now().UnixNano() // Fake ID
		createdAt = time.Now()
	}

	// Add to memory
	h.pg.AddSource(req.URL)

	// Trigger refresh in background with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.pg.Refresh(ctx); err != nil {
			log.Printf("Background proxy refresh failed: %v", err)
		}
	}()

	RenderJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         id,
		"url":        req.URL,
		"created_at": createdAt,
	})
}

func (h *ProxyHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		RenderError(w, http.StatusServiceUnavailable, "ProxyGate disabled")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		RenderError(w, http.StatusBadRequest, "ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid ID format")
		return
	}

	if h.repo != nil {
		// Get URL first to remove from memory
		source, err := h.repo.GetByID(r.Context(), id)
		if err != nil {
			RenderError(w, http.StatusNotFound, "Source not found")
			return
		}

		if err := h.repo.Delete(r.Context(), id); err != nil {
			log.Printf("Failed to delete proxy source: %v", err)
			RenderError(w, http.StatusInternalServerError, "Failed to delete proxy source")
			return
		}

		h.pg.RemoveSource(source.URL)
	} else {
		RenderError(w, http.StatusNotImplemented, "Persistence required for deletion")
		return
	}

	RenderJSON(w, http.StatusOK, map[string]string{"message": "Source deleted"})
}

func (h *ProxyHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	RenderError(w, http.StatusNotImplemented, "Not implemented yet")
}

// ListProxies returns paginated list of proxies from database
func (h *ProxyHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		RenderJSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]interface{}{
				"total": 0,
				"page":  1,
				"limit": 50,
			},
		})
		return
	}

	// Parse query params
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	status := query.Get("status")
	country := query.Get("country")

	params := domain.ProxyListParams{
		Limit:  limit,
		Offset: offset,
	}
	if status != "" {
		params.Status = domain.ProxyStatus(status)
	}
	if country != "" {
		params.Country = country
	}

	proxies, total, err := h.proxyListRepo.List(r.Context(), params)
	if err != nil {
		log.Printf("Failed to list proxies: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to list proxies")
		return
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"data": proxies,
		"meta": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetProxyStats returns proxy statistics from database
func (h *ProxyHandler) GetProxyStats(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		// Fallback to ProxyGate stats
		h.GetStats(w, r)
		return
	}

	stats, err := h.proxyListRepo.GetStats(r.Context())
	if err != nil {
		log.Printf("Failed to get proxy stats: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to get proxy stats")
		return
	}

	// Also get last updated from ProxyGate if available
	lastUpdatedStr := "never"
	if h.pg != nil {
		_, _, lastUpdated := h.pg.GetStats()
		if !lastUpdated.IsZero() {
			lastUpdatedStr = lastUpdated.Format(time.RFC3339)
		}
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"total_proxies":   stats.Total,
			"healthy_proxies": stats.Healthy,
			"dead_proxies":    stats.Dead,
			"banned_proxies":  stats.Banned,
			"pending_proxies": stats.Pending,
			"avg_uptime":      stats.AvgUptime,
			"last_updated":    lastUpdatedStr,
		},
	})
}

// DeleteDeadProxies removes all dead proxies from database
func (h *ProxyHandler) DeleteDeadProxies(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		RenderError(w, http.StatusServiceUnavailable, "Proxy list repository not configured")
		return
	}

	count, err := h.proxyListRepo.DeleteDead(r.Context())
	if err != nil {
		log.Printf("Failed to delete dead proxies: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to delete dead proxies")
		return
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Dead proxies deleted",
		"count":   count,
	})
}

// AddProxy adds a single proxy to the database
func (h *ProxyHandler) AddProxy(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		RenderError(w, http.StatusServiceUnavailable, "Proxy list repository not configured")
		return
	}

	var req struct {
		IP       string `json:"ip"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Country  string `json:"country,omitempty"`
		Status   string `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.IP == "" || req.Port == 0 {
		RenderError(w, http.StatusBadRequest, "IP and port are required")
		return
	}

	if req.Protocol == "" {
		req.Protocol = "socks5"
	}

	status := domain.ProxyStatusHealthy
	if req.Status != "" {
		status = domain.ProxyStatus(req.Status)
	}

	proxy := &domain.Proxy{
		IP:       req.IP,
		Port:     req.Port,
		Protocol: req.Protocol,
		Country:  req.Country,
		Status:   status,
	}

	if err := h.proxyListRepo.Upsert(r.Context(), proxy); err != nil {
		log.Printf("Failed to add proxy: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to add proxy")
		return
	}

	RenderJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       proxy.ID,
		"ip":       proxy.IP,
		"port":     proxy.Port,
		"protocol": proxy.Protocol,
		"status":   proxy.Status,
	})
}

// AddProxiesBulk adds multiple proxies to the database
func (h *ProxyHandler) AddProxiesBulk(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		RenderError(w, http.StatusServiceUnavailable, "Proxy list repository not configured")
		return
	}

	var req struct {
		Proxies []struct {
			IP       string `json:"ip"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol,omitempty"`
			Country  string `json:"country,omitempty"`
		} `json:"proxies"`
		Status string `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Proxies) == 0 {
		RenderError(w, http.StatusBadRequest, "Proxies array is required")
		return
	}

	status := domain.ProxyStatusHealthy
	if req.Status != "" {
		status = domain.ProxyStatus(req.Status)
	}

	var proxies []*domain.Proxy
	for _, p := range req.Proxies {
		if p.IP == "" || p.Port == 0 {
			continue
		}
		protocol := p.Protocol
		if protocol == "" {
			protocol = "socks5"
		}
		proxies = append(proxies, &domain.Proxy{
			IP:       p.IP,
			Port:     p.Port,
			Protocol: protocol,
			Country:  p.Country,
			Status:   status,
		})
	}

	if len(proxies) == 0 {
		RenderError(w, http.StatusBadRequest, "No valid proxies provided")
		return
	}

	if err := h.proxyListRepo.UpsertBatch(r.Context(), proxies); err != nil {
		log.Printf("Failed to add proxies: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to add proxies")
		return
	}

	RenderJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Proxies added",
		"count":   len(proxies),
	})
}

// UpdateProxyStatus updates the status of a proxy
func (h *ProxyHandler) UpdateProxyStatus(w http.ResponseWriter, r *http.Request) {
	if h.proxyListRepo == nil {
		RenderError(w, http.StatusServiceUnavailable, "Proxy list repository not configured")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		RenderError(w, http.StatusBadRequest, "ID is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid ID format")
		return
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status == "" {
		RenderError(w, http.StatusBadRequest, "Status is required")
		return
	}

	// Update status via repository
	if err := h.proxyListRepo.UpdateStatus(r.Context(), id, domain.ProxyStatus(req.Status)); err != nil {
		log.Printf("Failed to update proxy status: %v", err)
		RenderError(w, http.StatusInternalServerError, "Failed to update proxy status")
		return
	}

	RenderJSON(w, http.StatusOK, map[string]string{"message": "Status updated"})
}
