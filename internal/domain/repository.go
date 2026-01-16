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

	// ListAll retrieves all results with pagination (global view)
	ListAll(ctx context.Context, limit, offset int) ([][]byte, int, error)

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

// ProxyRepository defines the interface for proxy source persistence
type ProxyRepository interface {
	// Create creates a new proxy source
	Create(ctx context.Context, url string) (*ProxySource, error)

	// Delete deletes a proxy source by ID
	Delete(ctx context.Context, id int64) error

	// List retrieves all proxy sources
	List(ctx context.Context) ([]*ProxySource, error)

	// GetByID retrieves a proxy source by ID
	GetByID(ctx context.Context, id int64) (*ProxySource, error)
}

// ProxyListRepository defines the interface for proxy list persistence
type ProxyListRepository interface {
	// Upsert creates or updates a proxy (based on IP:port unique constraint)
	Upsert(ctx context.Context, proxy *Proxy) error

	// UpsertBatch creates or updates multiple proxies
	UpsertBatch(ctx context.Context, proxies []*Proxy) error

	// GetByAddress retrieves a proxy by IP:port
	GetByAddress(ctx context.Context, ip string, port int) (*Proxy, error)

	// List retrieves proxies with optional filtering
	List(ctx context.Context, params ProxyListParams) ([]*Proxy, int, error)

	// ListHealthy retrieves all healthy proxies (for Pool)
	ListHealthy(ctx context.Context) ([]*Proxy, error)

	// UpdateStatus updates the status of a proxy
	UpdateStatus(ctx context.Context, id int64, status ProxyStatus) error

	// IncrementFailCount increments fail count and optionally marks as dead
	IncrementFailCount(ctx context.Context, id int64, maxFails int) error

	// IncrementSuccessCount increments success count
	IncrementSuccessCount(ctx context.Context, id int64) error

	// MarkUsed updates the last_used timestamp
	MarkUsed(ctx context.Context, id int64) error

	// DeleteDead removes all dead proxies
	DeleteDead(ctx context.Context) (int, error)

	// GetStats retrieves proxy statistics
	GetStats(ctx context.Context) (*ProxyStats, error)
}

// BusinessListingRepository defines the interface for business listing persistence
type BusinessListingRepository interface {
	// List retrieves business listings with filters and pagination
	List(ctx context.Context, filter BusinessListingFilter) ([]*BusinessListing, int, error)

	// ListByJobID retrieves business listings for a specific job
	ListByJobID(ctx context.Context, jobID string, limit, offset int) ([]*BusinessListing, int, error)

	// GetByID retrieves a single business listing by ID
	GetByID(ctx context.Context, id int64) (*BusinessListing, error)

	// GetCategories returns distinct categories
	GetCategories(ctx context.Context, limit int) ([]string, error)

	// GetCities returns distinct cities
	GetCities(ctx context.Context, limit int) ([]string, error)

	// Stats returns aggregate statistics
	Stats(ctx context.Context) (*BusinessListingStats, error)

	// Stream streams business listings for export
	Stream(ctx context.Context, filter BusinessListingFilter, fn func(listing *BusinessListing) error) error

	// StreamByJobID streams business listings for a specific job
	StreamByJobID(ctx context.Context, jobID string, fn func(listing *BusinessListing) error) error

	// CountByJobID counts business listings for a job
	CountByJobID(ctx context.Context, jobID string) (int, error)
}
