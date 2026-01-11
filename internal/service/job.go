package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/gosom/google-maps-scraper/internal/domain"
)

// Common errors
var (
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrJobNotPausable    = errors.New("job cannot be paused")
	ErrJobNotResumable   = errors.New("job cannot be resumed")
	ErrJobNotCancellable = errors.New("job cannot be cancelled")
)

// JobService handles job business logic
type JobService struct {
	jobs    domain.JobRepository
	results domain.ResultRepository
}

// NewJobService creates a new JobService
func NewJobService(jobs domain.JobRepository, results domain.ResultRepository) *JobService {
	return &JobService{
		jobs:    jobs,
		results: results,
	}
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, req *domain.CreateJobRequest) (*domain.Job, error) {
	job := req.ToJob()

	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetByID retrieves a job by ID
func (s *JobService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	return job, nil
}

// List retrieves jobs with optional filtering
func (s *JobService) List(ctx context.Context, params domain.JobListParams) ([]*domain.Job, int, error) {
	jobs, total, err := s.jobs.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, total, nil
}

// Delete deletes a job and its results
func (s *JobService) Delete(ctx context.Context, id uuid.UUID) error {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return ErrJobNotFound
	}

	// Don't allow deleting running jobs
	if job.Status == domain.JobStatusRunning {
		return errors.New("cannot delete a running job, cancel it first")
	}

	// Delete results first (cascade should handle this, but be explicit)
	if err := s.results.DeleteByJobID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete results: %w", err)
	}

	if err := s.jobs.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// Pause pauses a running or queued job
func (s *JobService) Pause(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	if !job.Status.CanPause() {
		return nil, ErrJobNotPausable
	}

	if err := s.jobs.UpdateStatus(ctx, id, domain.JobStatusPaused); err != nil {
		return nil, fmt.Errorf("failed to pause job: %w", err)
	}

	job.Status = domain.JobStatusPaused
	return job, nil
}

// Resume resumes a paused job
func (s *JobService) Resume(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	if !job.Status.CanResume() {
		return nil, ErrJobNotResumable
	}

	// Resume to pending so a worker can pick it up
	if err := s.jobs.UpdateStatus(ctx, id, domain.JobStatusPending); err != nil {
		return nil, fmt.Errorf("failed to resume job: %w", err)
	}

	job.Status = domain.JobStatusPending
	return job, nil
}

// Cancel cancels a job
func (s *JobService) Cancel(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	if !job.Status.CanCancel() {
		return nil, ErrJobNotCancellable
	}

	if err := s.jobs.UpdateStatus(ctx, id, domain.JobStatusCancelled); err != nil {
		return nil, fmt.Errorf("failed to cancel job: %w", err)
	}

	job.Status = domain.JobStatusCancelled
	return job, nil
}

// UpdateProgress updates job progress
func (s *JobService) UpdateProgress(ctx context.Context, id uuid.UUID, progress domain.JobProgress) error {
	progress.CalculatePercentage()
	return s.jobs.UpdateProgress(ctx, id, progress)
}

// Complete marks a job as completed
func (s *JobService) Complete(ctx context.Context, id uuid.UUID) error {
	return s.jobs.UpdateStatus(ctx, id, domain.JobStatusCompleted)
}

// Fail marks a job as failed with an error message
func (s *JobService) Fail(ctx context.Context, id uuid.UUID, errMsg string) error {
	job, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFound
	}

	job.Status = domain.JobStatusFailed
	job.ErrorMessage = &errMsg

	return s.jobs.Update(ctx, job)
}

// GetStats retrieves job statistics
func (s *JobService) GetStats(ctx context.Context) (*domain.JobStats, error) {
	return s.jobs.GetStats(ctx)
}
