package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/proxygate"
)

type ProxyHandler struct {
	pg   *proxygate.ProxyGate
	repo domain.ProxyRepository
}

func NewProxyHandler(pg *proxygate.ProxyGate, repo domain.ProxyRepository) *ProxyHandler {
	return &ProxyHandler{pg: pg, repo: repo}
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
			RenderError(w, http.StatusInternalServerError, err.Error())
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
		RenderError(w, http.StatusInternalServerError, err.Error())
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

	// Persist
	var id int64
	var createdAt time.Time

	if h.repo != nil {
		source, err := h.repo.Create(r.Context(), req.URL)
		if err != nil {
			RenderError(w, http.StatusInternalServerError, err.Error())
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

	// Trigger refresh in background
	go h.pg.Refresh(context.Background())

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
			RenderError(w, http.StatusInternalServerError, err.Error())
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
