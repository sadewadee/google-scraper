package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/scrapemate"

	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/mq"
	"github.com/sadewadee/google-scraper/internal/queue"
	"github.com/sadewadee/google-scraper/internal/spawner"
	"github.com/sadewadee/google-scraper/postgres"
	"github.com/sadewadee/google-scraper/runner"
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
	jobs      domain.JobRepository
	results   domain.ResultRepository
	queue     *queue.Queue       // Redis queue (legacy)
	mqPub     mq.Publisher       // RabbitMQ publisher (preferred)
	gmapsPush postgres.GmapsJobPusher // Bridge to gmaps_jobs for DSN workers
	spawner   spawner.Spawner    // Auto-spawn workers on job creation
}

// NewJobService creates a new JobService
func NewJobService(jobs domain.JobRepository, results domain.ResultRepository, q *queue.Queue) *JobService {
	return &JobService{
		jobs:    jobs,
		results: results,
		queue:   q,
	}
}

// NewJobServiceWithBridge creates a new JobService with DSN bridge support.
// The gmapsPush parameter enables bridging Dashboard jobs to gmaps_jobs table
// so DSN workers can pick them up.
func NewJobServiceWithBridge(jobs domain.JobRepository, results domain.ResultRepository, q *queue.Queue, gmapsPush postgres.GmapsJobPusher) *JobService {
	return &JobService{
		jobs:      jobs,
		results:   results,
		queue:     q,
		gmapsPush: gmapsPush,
	}
}

// NewJobServiceWithMQ creates a new JobService with RabbitMQ support.
// This is the preferred constructor for Manager mode with RabbitMQ.
func NewJobServiceWithMQ(jobs domain.JobRepository, results domain.ResultRepository, mqPub mq.Publisher, gmapsPush postgres.GmapsJobPusher) *JobService {
	return &JobService{
		jobs:      jobs,
		results:   results,
		mqPub:     mqPub,
		gmapsPush: gmapsPush,
	}
}

// NewJobServiceWithSpawner creates a new JobService with auto-spawn support.
// This is the recommended constructor for production Manager mode.
func NewJobServiceWithSpawner(jobs domain.JobRepository, results domain.ResultRepository, mqPub mq.Publisher, gmapsPush postgres.GmapsJobPusher, sp spawner.Spawner) *JobService {
	return &JobService{
		jobs:      jobs,
		results:   results,
		mqPub:     mqPub,
		gmapsPush: gmapsPush,
		spawner:   sp,
	}
}

// SetSpawner sets the worker spawner (can be called after construction)
func (s *JobService) SetSpawner(sp spawner.Spawner) {
	s.spawner = sp
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, req *domain.CreateJobRequest) (*domain.Job, error) {
	start := time.Now()
	log.Printf("[JobService] Create started for job: %s", req.Name)

	job := req.ToJob()
	log.Printf("[JobService] ToJob completed in %v", time.Since(start))

	dbStart := time.Now()
	if err := s.jobs.Create(ctx, job); err != nil {
		log.Printf("[JobService] Create FAILED after %v: %v", time.Since(start), err)
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("[JobService] Create completed in %v (db: %v)", time.Since(start), time.Since(dbStart))

	// Bridge to gmaps_jobs for DSN workers (if configured)
	if s.gmapsPush != nil {
		bridgeStart := time.Now()
		if err := s.bridgeToGmapsJobs(ctx, job); err != nil {
			log.Printf("[JobService] WARNING: bridge to gmaps_jobs failed for job %s: %v", job.ID, err)
			// Don't fail job creation - just log the error
			// The Redis queue fallback can still work
		} else {
			log.Printf("[JobService] Job %s bridged to gmaps_jobs (%d tasks) in %v",
				job.ID, job.Progress.TotalPlaces, time.Since(bridgeStart))
			// Update job with total tasks count from bridge
			if err := s.jobs.Update(ctx, job); err != nil {
				log.Printf("[JobService] WARNING: failed to update total_tasks for job %s: %v", job.ID, err)
			}
		}
	}

	// Enqueue to RabbitMQ if available (preferred over Redis)
	if s.mqPub != nil {
		msg := &mq.JobMessage{
			JobID:    job.ID,
			Priority: job.Priority,
			Type:     "job:process",
		}
		if err := s.mqPub.Publish(ctx, msg); err != nil {
			log.Printf("[JobService] WARNING: failed to publish job %s to RabbitMQ: %v", job.ID, err)
		} else {
			log.Printf("[JobService] Job %s published to RabbitMQ queue", job.ID)
		}
	} else if s.queue != nil {
		// Fallback to Redis queue if RabbitMQ not available
		if err := s.queue.Enqueue(ctx, job.ID, job.Priority); err != nil {
			// Log error but don't fail job creation - worker can still poll
			log.Printf("[JobService] WARNING: failed to enqueue job %s to Redis: %v", job.ID, err)
		} else {
			log.Printf("[JobService] Job %s enqueued to Redis queue", job.ID)
		}
	}

	// Auto-spawn worker if spawner is configured
	if s.spawner != nil {
		go s.spawnWorkerForJob(job)
	}

	return job, nil
}

// bridgeToGmapsJobs creates seed jobs and inserts them into gmaps_jobs table.
// This bridges the Dashboard job (jobs_queue) to DSN workers (gmaps_jobs).
func (s *JobService) bridgeToGmapsJobs(ctx context.Context, job *domain.Job) error {
	var allSeedJobs []scrapemate.IJob
	parentID := job.ID.String()

	// Check if full coverage mode is enabled with valid bounding box
	if job.Config.CoverageMode == domain.CoverageModeFull &&
		job.Config.BoundingBox != nil &&
		job.Config.BoundingBox.IsValid() {

		// Generate grid points from bounding box based on radius
		radius := job.Config.Radius
		if radius < 100 {
			radius = 5000 // default 5km
		}
		gridPoints := job.Config.BoundingBox.GenerateGridByRadius(radius)

		log.Printf("[JobService] Full coverage mode enabled: generating %d grid points for job %s (radius: %dm)",
			len(gridPoints), job.ID, radius)

		// Create seed jobs for each grid point
		for i, point := range gridPoints {
			geoCoords := runner.FormatGeoCoordinates(point.Lat, point.Lon)

			seedJobs, err := runner.CreateSeedJobsFromKeywords(runner.SeedJobConfig{
				Keywords:       job.Config.Keywords,
				FastMode:       job.Config.FastMode,
				LangCode:       job.Config.Lang,
				Depth:          job.Config.Depth,
				Email:          job.Config.ExtractEmail,
				GeoCoordinates: geoCoords,
				Zoom:           job.Config.Zoom,
				Radius:         float64(job.Config.Radius),
				ExtraReviews:   false, // Not exposed in Dashboard yet
				Dedup:          nil,   // Deduplication handled by workers
				ExitMonitor:    nil,   // Not needed for bridge
			})
			if err != nil {
				return fmt.Errorf("failed to create seed jobs for grid point %d (%.4f, %.4f): %w",
					i, point.Lat, point.Lon, err)
			}

			allSeedJobs = append(allSeedJobs, seedJobs...)
		}

		log.Printf("[JobService] Full coverage mode: created %d total seed jobs for job %s",
			len(allSeedJobs), job.ID)
	} else {
		// Single point mode (default/legacy behavior)
		geoCoords := ""
		if job.Config.GeoLat != nil && job.Config.GeoLon != nil {
			geoCoords = runner.FormatGeoCoordinates(*job.Config.GeoLat, *job.Config.GeoLon)
		}

		seedJobs, err := runner.CreateSeedJobsFromKeywords(runner.SeedJobConfig{
			Keywords:       job.Config.Keywords,
			FastMode:       job.Config.FastMode,
			LangCode:       job.Config.Lang,
			Depth:          job.Config.Depth,
			Email:          job.Config.ExtractEmail,
			GeoCoordinates: geoCoords,
			Zoom:           job.Config.Zoom,
			Radius:         float64(job.Config.Radius),
			ExtraReviews:   false, // Not exposed in Dashboard yet
			Dedup:          nil,   // Deduplication handled by workers
			ExitMonitor:    nil,   // Not needed for bridge
		})
		if err != nil {
			return fmt.Errorf("failed to create seed jobs: %w", err)
		}

		allSeedJobs = seedJobs
	}

	// Push each seed job to gmaps_jobs with parent reference
	for _, seedJob := range allSeedJobs {
		if err := s.gmapsPush.PushWithParent(ctx, seedJob, parentID); err != nil {
			return fmt.Errorf("failed to push seed job %s: %w", seedJob.GetID(), err)
		}
	}

	// Update job with total tasks count
	job.Progress.TotalPlaces = len(allSeedJobs)

	return nil
}

// spawnWorkerForJob spawns a worker container/function to process the job
func (s *JobService) spawnWorkerForJob(job *domain.Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &spawner.SpawnRequest{
		JobID:       job.ID,
		Priority:    job.Priority,
		Concurrency: 4, // Default concurrency
	}

	result, err := s.spawner.Spawn(ctx, req)
	if err != nil {
		log.Printf("[JobService] WARNING: failed to spawn worker for job %s: %v", job.ID, err)
		return
	}

	if result.Error != "" {
		log.Printf("[JobService] WARNING: spawner returned error for job %s: %s", job.ID, result.Error)
		return
	}

	log.Printf("[JobService] Spawned worker for job %s (worker: %s, status: %s, spawner: %s)",
		job.ID, result.WorkerID, result.Status, s.spawner.Name())
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

	// Re-enqueue to RabbitMQ if available (preferred over Redis)
	if s.mqPub != nil {
		msg := &mq.JobMessage{
			JobID:    job.ID,
			Priority: job.Priority,
			Type:     "job:process",
		}
		if err := s.mqPub.Publish(ctx, msg); err != nil {
			log.Printf("[JobService] WARNING: failed to re-publish resumed job %s to RabbitMQ: %v", job.ID, err)
		} else {
			log.Printf("[JobService] Resumed job %s re-published to RabbitMQ queue", job.ID)
		}
	} else if s.queue != nil {
		// Fallback to Redis queue
		if err := s.queue.Enqueue(ctx, job.ID, job.Priority); err != nil {
			log.Printf("[JobService] WARNING: failed to re-enqueue resumed job %s to Redis: %v", job.ID, err)
		} else {
			log.Printf("[JobService] Resumed job %s re-enqueued to Redis queue", job.ID)
		}
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
