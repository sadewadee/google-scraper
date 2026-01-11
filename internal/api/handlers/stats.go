package handlers

import (
	"context"
	"net/http"

	"github.com/gosom/google-maps-scraper/internal/domain"
)

// StatsServiceInterface defines the stats service methods
type StatsServiceInterface interface {
	GetStats(ctx context.Context) (*domain.Stats, error)
}

// StatsHandler handles statistics-related HTTP requests
type StatsHandler struct {
	stats StatsServiceInterface
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(stats StatsServiceInterface) *StatsHandler {
	return &StatsHandler{
		stats: stats,
	}
}

// GetDashboardStats handles GET /api/v2/stats
func (h *StatsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats, err := h.stats.GetStats(r.Context())
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	RenderJSON(w, http.StatusOK, stats)
}
