package domain

import (
	"time"

	"github.com/google/uuid"
)

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusOffline WorkerStatus = "offline"
)

// Worker represents a scraper worker instance
type Worker struct {
	ID             string       `json:"id"`
	Hostname       string       `json:"hostname"`
	Status         WorkerStatus `json:"status"`
	CurrentJobID   *uuid.UUID   `json:"current_job_id,omitempty"`
	CurrentJobName *string      `json:"current_job_name,omitempty"` // Populated via join

	// Stats
	JobsCompleted int `json:"jobs_completed"`
	PlacesScraped int `json:"places_scraped"`

	// Heartbeat
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
}

// IsOnline returns true if the worker has sent a heartbeat recently
func (w *Worker) IsOnline(timeout time.Duration) bool {
	return time.Since(w.LastHeartbeat) < timeout
}

// WorkerHeartbeat is the request from a worker to update its status
type WorkerHeartbeat struct {
	WorkerID     string       `json:"worker_id"`
	Hostname     string       `json:"hostname"`
	Status       WorkerStatus `json:"status"`
	CurrentJobID *uuid.UUID   `json:"current_job_id,omitempty"`
}

// WorkerStats contains aggregated worker statistics
type WorkerStats struct {
	TotalWorkers  int `json:"total_workers"`
	OnlineWorkers int `json:"online_workers"`
	BusyWorkers   int `json:"busy_workers"`
	IdleWorkers   int `json:"idle_workers"`
}

// WorkerListParams are parameters for listing workers
type WorkerListParams struct {
	Status *WorkerStatus
	Limit  int
	Offset int
}

// HeartbeatTimeout is the duration after which a worker is considered offline
const HeartbeatTimeout = 30 * time.Second

// HeartbeatInterval is how often workers should send heartbeats
const HeartbeatInterval = 10 * time.Second
