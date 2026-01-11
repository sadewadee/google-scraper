package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/gosom/google-maps-scraper/internal/domain"
)

// WorkerService handles worker business logic
type WorkerService struct {
	workers domain.WorkerRepository
	jobs    domain.JobRepository
}

// NewWorkerService creates a new WorkerService
func NewWorkerService(workers domain.WorkerRepository, jobs domain.JobRepository) *WorkerService {
	return &WorkerService{
		workers: workers,
		jobs:    jobs,
	}
}

// Register registers a new worker or updates existing one
func (s *WorkerService) Register(ctx context.Context, workerID string) (*domain.Worker, error) {
	hostname, _ := os.Hostname()

	worker := &domain.Worker{
		ID:            workerID,
		Hostname:      hostname,
		Status:        domain.WorkerStatusIdle,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.workers.Upsert(ctx, worker); err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	return worker, nil
}

// Heartbeat updates worker heartbeat and status
func (s *WorkerService) Heartbeat(ctx context.Context, hb *domain.WorkerHeartbeat) error {
	hostname := hb.Hostname
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	worker := &domain.Worker{
		ID:            hb.WorkerID,
		Hostname:      hostname,
		Status:        hb.Status,
		CurrentJobID:  hb.CurrentJobID,
		LastHeartbeat: time.Now().UTC(),
	}

	return s.workers.Upsert(ctx, worker)
}

// List retrieves all workers
func (s *WorkerService) List(ctx context.Context, params domain.WorkerListParams) ([]*domain.Worker, error) {
	return s.workers.List(ctx, params)
}

// GetByID retrieves a worker by ID
func (s *WorkerService) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	return s.workers.GetByID(ctx, id)
}

// GetStats retrieves worker statistics
func (s *WorkerService) GetStats(ctx context.Context) (*domain.WorkerStats, error) {
	return s.workers.GetStats(ctx)
}

// ClaimJob claims a pending job for a worker
func (s *WorkerService) ClaimJob(ctx context.Context, workerID string) (*domain.Job, error) {
	// Update worker status to busy
	job, err := s.jobs.ClaimJob(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	if job == nil {
		return nil, nil // No pending jobs
	}

	// Update worker status
	if err := s.workers.UpdateStatus(ctx, workerID, domain.WorkerStatusBusy); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to update worker status: %v\n", err)
	}

	return job, nil
}

// ReleaseJob releases a job back to pending (e.g., worker crashed)
func (s *WorkerService) ReleaseJob(ctx context.Context, jobID uuid.UUID, workerID string) error {
	if err := s.jobs.ReleaseJob(ctx, jobID); err != nil {
		return fmt.Errorf("failed to release job: %w", err)
	}

	// Update worker status to idle
	if err := s.workers.UpdateStatus(ctx, workerID, domain.WorkerStatusIdle); err != nil {
		fmt.Printf("warning: failed to update worker status: %v\n", err)
	}

	return nil
}

// CompleteJob marks job as completed and updates worker stats
func (s *WorkerService) CompleteJob(ctx context.Context, jobID uuid.UUID, workerID string, placesScraped int) error {
	// Mark job as completed
	if err := s.jobs.UpdateStatus(ctx, jobID, domain.JobStatusCompleted); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	// Update worker stats and status
	if err := s.workers.IncrementStats(ctx, workerID, 1, placesScraped); err != nil {
		fmt.Printf("warning: failed to update worker stats: %v\n", err)
	}

	if err := s.workers.UpdateStatus(ctx, workerID, domain.WorkerStatusIdle); err != nil {
		fmt.Printf("warning: failed to update worker status: %v\n", err)
	}

	return nil
}

// FailJob marks job as failed and updates worker
func (s *WorkerService) FailJob(ctx context.Context, jobID uuid.UUID, workerID string, errMsg string) error {
	// Get job to update with error message
	job, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	job.Status = domain.JobStatusFailed
	job.ErrorMessage = &errMsg

	if err := s.jobs.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update worker status to idle
	if err := s.workers.UpdateStatus(ctx, workerID, domain.WorkerStatusIdle); err != nil {
		fmt.Printf("warning: failed to update worker status: %v\n", err)
	}

	return nil
}

// MarkOfflineWorkers marks stale workers as offline and releases their jobs
func (s *WorkerService) MarkOfflineWorkers(ctx context.Context) (int, error) {
	timeout := int(domain.HeartbeatTimeout.Seconds())
	return s.workers.MarkOfflineWorkers(ctx, timeout)
}

// Unregister removes a worker
func (s *WorkerService) Unregister(ctx context.Context, workerID string) error {
	// First check if worker has a job
	worker, err := s.workers.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	if worker != nil && worker.CurrentJobID != nil {
		// Release the job back to pending
		if err := s.jobs.ReleaseJob(ctx, *worker.CurrentJobID); err != nil {
			fmt.Printf("warning: failed to release job: %v\n", err)
		}
	}

	return s.workers.Delete(ctx, workerID)
}
