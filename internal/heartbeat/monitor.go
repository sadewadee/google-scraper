package heartbeat

import (
	"context"
	"log"
	"time"

	"github.com/gosom/google-maps-scraper/internal/domain"
)

// WorkerService defines methods needed for heartbeat monitoring
type WorkerService interface {
	MarkOfflineWorkers(ctx context.Context) (int, error)
}

// Monitor monitors worker heartbeats and marks stale workers as offline
type Monitor struct {
	workers  WorkerService
	interval time.Duration
}

// NewMonitor creates a new heartbeat monitor
func NewMonitor(workers WorkerService, interval time.Duration) *Monitor {
	if interval == 0 {
		interval = domain.HeartbeatInterval
	}

	return &Monitor{
		workers:  workers,
		interval: interval,
	}
}

// Run starts the heartbeat monitor
func (m *Monitor) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	log.Printf("heartbeat monitor started (interval: %s, timeout: %s)",
		m.interval, domain.HeartbeatTimeout)

	for {
		select {
		case <-ctx.Done():
			log.Println("heartbeat monitor stopped")
			return nil
		case <-ticker.C:
			count, err := m.workers.MarkOfflineWorkers(ctx)
			if err != nil {
				log.Printf("error marking offline workers: %v", err)
				continue
			}

			if count > 0 {
				log.Printf("marked %d workers as offline", count)
			}
		}
	}
}
