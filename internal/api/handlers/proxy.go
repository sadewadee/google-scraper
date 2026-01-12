package handlers

import (
	"net/http"

	"github.com/sadewadee/google-scraper/internal/proxygate"
)

type ProxyHandler struct {
	pg *proxygate.ProxyGate
}

func NewProxyHandler(pg *proxygate.ProxyGate) *ProxyHandler {
	return &ProxyHandler{pg: pg}
}

func (h *ProxyHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		RenderError(w, http.StatusNotFound, "ProxyGate not enabled")
		return
	}

	total, healthy := h.pg.GetStats()

	stats := map[string]interface{}{
		"total_proxies":   total,
		"healthy_proxies": healthy,
		"last_updated":    "now", // TODO: Add last update time
	}

	RenderJSON(w, http.StatusOK, map[string]interface{}{
		"data": stats,
	})
}

func (h *ProxyHandler) GetSources(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		RenderError(w, http.StatusNotFound, "ProxyGate not enabled")
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
		RenderError(w, http.StatusNotFound, "ProxyGate not enabled")
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
	RenderError(w, http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProxyHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	RenderError(w, http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProxyHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	RenderError(w, http.StatusNotImplemented, "Not implemented yet")
}
