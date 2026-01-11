package domain

import (
	"context"

	"github.com/google/uuid"
)

// JobRepository defines the interface for job persistence
type JobRepository interface {
	// Create creates a new job
	Create(ctx context.Context, job *Job) error

	// GetByID retrieves a job by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)

	// List retrieves jobs with optional filtering
	List(ctx context.Context, params JobListParams) ([]*Job, int, error)

	// Update updates a job
	Update(ctx context.Context, job *Job) error

	// Delete deletes a job by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateStatus updates only the status of a job
	UpdateStatus(ctx context.Context, id uuid.UUID, status JobStatus) error

	// UpdateProgress updates the progress of a job
	UpdateProgress(ctx context.Context, id uuid.UUID, progress JobProgress) error

	// ClaimJob claims a pending job for a worker (atomic operation)
	ClaimJob(ctx context.Context, workerID string) (*Job, error)

	// ReleaseJob releases a job back to pending status
	ReleaseJob(ctx context.Context, id uuid.UUID) error

	// GetStats retrieves job statistics
	GetStats(ctx context.Context) (*JobStats, error)
}

// WorkerRepository defines the interface for worker persistence
type WorkerRepository interface {
	// Upsert creates or updates a worker (for heartbeat)
	Upsert(ctx context.Context, worker *Worker) error

	// GetByID retrieves a worker by ID
	GetByID(ctx context.Context, id string) (*Worker, error)

	// List retrieves all workers
	List(ctx context.Context, params WorkerListParams) ([]*Worker, error)

	// Delete deletes a worker by ID
	Delete(ctx context.Context, id string) error

	// UpdateStatus updates only the status of a worker
	UpdateStatus(ctx context.Context, id string, status WorkerStatus) error

	// MarkOfflineWorkers marks workers as offline if heartbeat is stale
	MarkOfflineWorkers(ctx context.Context, timeout int) (int, error)

	// GetStats retrieves worker statistics
	GetStats(ctx context.Context) (*WorkerStats, error)

	// IncrementStats increments worker statistics
	IncrementStats(ctx context.Context, id string, jobsCompleted, placesScraped int) error
}

// ResultRepository defines the interface for result persistence
type ResultRepository interface {
	// Create creates a new result
	Create(ctx context.Context, jobID uuid.UUID, data []byte) error

	// CreateBatch creates multiple results in a batch
	CreateBatch(ctx context.Context, jobID uuid.UUID, data [][]byte) error

	// ListByJobID retrieves results for a job with pagination
	ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error)

	// CountByJobID counts results for a job
	CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error)

	// DeleteByJobID deletes all results for a job
	DeleteByJobID(ctx context.Context, jobID uuid.UUID) error

	// GetPlaceStats retrieves place statistics
	GetPlaceStats(ctx context.Context) (*PlaceStats, error)

	// StreamByJobID streams results for a job (memory efficient)
	StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error
}
