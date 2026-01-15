package worker

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/deduper"
	"github.com/sadewadee/google-scraper/exiter"
	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/emailvalidator"
	"github.com/sadewadee/google-scraper/internal/mq"
	"github.com/sadewadee/google-scraper/internal/queue"
	"github.com/sadewadee/google-scraper/runner"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/adapters/writers/csvwriter"
	"github.com/gosom/scrapemate/scrapemateapp"
)

// Config holds worker configuration
type Config struct {
	ManagerURL   string
	WorkerID     string
	RunnerConfig *runner.Config
	RedisURL     string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	RabbitMQURL  string // RabbitMQ URL (preferred over Redis for job queue)
}

// Runner is a worker that claims and processes jobs from the manager
type Runner struct {
	client       *Client
	config       *runner.Config
	dataFolder   string
	currentJob   *domain.Job
	jobMu        sync.RWMutex // Protects currentJob
	stopChan     chan struct{}
	stopOnce     sync.Once
	workerID     string
	queueWorker  *queue.Worker
	useRedis     bool
	redisDeduper *queue.Deduper
	mqConsumer   *mq.RabbitMQConsumer
	useRabbitMQ  bool
}

// NewRunner creates a new worker runner
func NewRunner(cfg *Config) (*Runner, error) {
	if cfg.RunnerConfig == nil {
		cfg.RunnerConfig = &runner.Config{}
	}

	if cfg.RunnerConfig.DataFolder == "" {
		cfg.RunnerConfig.DataFolder = "."
	}

	if err := os.MkdirAll(cfg.RunnerConfig.DataFolder, os.ModePerm); err != nil {
		return nil, err
	}

	r := &Runner{
		client:      NewClient(cfg.ManagerURL, cfg.WorkerID),
		config:      cfg.RunnerConfig,
		dataFolder:  cfg.RunnerConfig.DataFolder,
		workerID:    cfg.WorkerID,
		stopChan:    make(chan struct{}),
		useRedis:    false,
		useRabbitMQ: false,
	}

	// Try to set up RabbitMQ consumer (preferred over Redis for job queue)
	if cfg.RabbitMQURL != "" {
		consumerCfg := mq.ConsumerConfig{
			URL:        cfg.RabbitMQURL,
			Prefetch:   1, // Process one job at a time per worker
			ConsumerID: cfg.WorkerID,
		}

		consumer, err := mq.NewConsumer(consumerCfg)
		if err != nil {
			log.Printf("WARNING: failed to connect to RabbitMQ: %v", err)
			log.Println("falling back to Redis queue or HTTP polling mode")
		} else {
			r.mqConsumer = consumer
			r.useRabbitMQ = true
			log.Println("RabbitMQ consumer initialized")
		}
	}

	// Try to set up Redis queue worker and deduper (fallback for job queue, always for dedup)
	if cfg.RedisURL != "" || cfg.RedisAddr != "" {
		// Initialize Redis queue worker
		queueCfg := &queue.WorkerConfig{
			RedisURL:    cfg.RedisURL,
			RedisAddr:   cfg.RedisAddr,
			Password:    cfg.RedisPass,
			DB:          cfg.RedisDB,
			Concurrency: 1, // Process one job at a time per worker
		}

		qw, err := queue.NewWorker(queueCfg, r.handleQueueJob)
		if err != nil {
			log.Printf("WARNING: failed to connect to Redis queue: %v", err)
			log.Println("falling back to HTTP polling mode")
		} else {
			r.queueWorker = qw
			r.useRedis = true
			log.Println("Redis queue worker initialized")
		}

		// Initialize Redis deduper for distributed deduplication
		dedupCfg := &queue.DedupeConfig{
			RedisURL:  cfg.RedisURL,
			RedisAddr: cfg.RedisAddr,
			Password:  cfg.RedisPass,
			DB:        cfg.RedisDB,
			Prefix:    "dedup",
		}

		dedup, err := queue.NewDeduper(dedupCfg)
		if err != nil {
			log.Printf("WARNING: failed to connect to Redis deduper: %v", err)
			log.Println("using local in-memory deduplication")
		} else {
			r.redisDeduper = dedup
			log.Println("Redis distributed deduplication initialized")
		}
	}

	return r, nil
}

// setCurrentJob safely sets the current job with locking
func (r *Runner) setCurrentJob(job *domain.Job) {
	r.jobMu.Lock()
	r.currentJob = job
	r.jobMu.Unlock()
}

// getCurrentJob safely gets the current job with locking
func (r *Runner) getCurrentJob() *domain.Job {
	r.jobMu.RLock()
	defer r.jobMu.RUnlock()
	return r.currentJob
}

// Run starts the worker
func (r *Runner) Run(ctx context.Context) error {
	// Register with manager
	worker, err := r.client.Register(ctx)
	if err != nil {
		return err
	}

	log.Printf("worker registered: %s (hostname: %s)", worker.ID, worker.Hostname)

	// Start heartbeat goroutine
	go r.heartbeatLoop(ctx)

	// Use RabbitMQ if available (preferred), then Redis, then HTTP polling
	if r.useRabbitMQ && r.mqConsumer != nil {
		log.Println("starting RabbitMQ consumer mode")
		return r.mqConsumer.Consume(ctx, r.handleMQJob)
	}

	if r.useRedis && r.queueWorker != nil {
		log.Println("starting Redis queue worker mode")
		return r.queueWorker.Run(ctx)
	}

	log.Println("starting HTTP polling mode")
	return r.workLoop(ctx)
}

// Stop gracefully stops the worker
func (r *Runner) Stop(ctx context.Context) error {
	r.stopOnce.Do(func() {
		close(r.stopChan)
	})

	// Close RabbitMQ consumer if active
	if r.mqConsumer != nil {
		r.mqConsumer.Close()
	}

	// Shutdown Redis queue worker if active
	if r.queueWorker != nil {
		r.queueWorker.Shutdown()
	}

	// Close Redis deduper if active
	if r.redisDeduper != nil {
		r.redisDeduper.Close()
	}

	// Release current job if any
	if currentJob := r.getCurrentJob(); currentJob != nil {
		if err := r.client.ReleaseJob(ctx, currentJob.ID); err != nil {
			log.Printf("warning: failed to release job: %v", err)
		}
	}

	// Unregister from manager
	if err := r.client.Unregister(ctx); err != nil {
		log.Printf("warning: failed to unregister: %v", err)
	}

	return nil
}

// handleMQJob is called by the RabbitMQ consumer for each job
func (r *Runner) handleMQJob(ctx context.Context, msg *mq.JobMessage) error {
	log.Printf("received job from RabbitMQ: %s", msg.JobID)

	// Fetch full job details from manager
	job, err := r.fetchJobDetails(ctx, msg.JobID)
	if err != nil {
		log.Printf("failed to fetch job details: %v", err)
		return err
	}

	if job == nil {
		log.Printf("job %s not found or already completed", msg.JobID)
		return nil // Not an error, job may have been cancelled
	}

	r.setCurrentJob(job)

	// Process the job
	placesScraped, err := r.processJob(ctx, job)
	if err != nil {
		log.Printf("job failed: %s - %v", job.ID, err)
		if failErr := r.client.FailJob(ctx, job.ID, err.Error()); failErr != nil {
			log.Printf("warning: failed to mark job as failed: %v", failErr)
		}
		r.setCurrentJob(nil)
		return err
	}

	log.Printf("job completed: %s (%d places)", job.ID, placesScraped)
	if completeErr := r.client.CompleteJob(ctx, job.ID, placesScraped); completeErr != nil {
		log.Printf("warning: failed to mark job as completed: %v", completeErr)
	}

	r.setCurrentJob(nil)
	return nil
}

// handleQueueJob is called by the Redis queue worker for each job
func (r *Runner) handleQueueJob(ctx context.Context, payload *queue.JobPayload) error {
	log.Printf("received job from Redis queue: %s", payload.JobID)

	// Fetch full job details from manager
	job, err := r.fetchJobDetails(ctx, payload.JobID)
	if err != nil {
		log.Printf("failed to fetch job details: %v", err)
		return err
	}

	if job == nil {
		log.Printf("job %s not found or already completed", payload.JobID)
		return nil // Not an error, job may have been cancelled
	}

	r.setCurrentJob(job)

	// Process the job
	placesScraped, err := r.processJob(ctx, job)
	if err != nil {
		log.Printf("job failed: %s - %v", job.ID, err)
		if failErr := r.client.FailJob(ctx, job.ID, err.Error()); failErr != nil {
			log.Printf("warning: failed to mark job as failed: %v", failErr)
		}
		r.setCurrentJob(nil)
		return err
	}

	log.Printf("job completed: %s (%d places)", job.ID, placesScraped)
	if completeErr := r.client.CompleteJob(ctx, job.ID, placesScraped); completeErr != nil {
		log.Printf("warning: failed to mark job as completed: %v", completeErr)
	}

	r.setCurrentJob(nil)
	return nil
}

// fetchJobDetails fetches full job details from the manager API by job ID
func (r *Runner) fetchJobDetails(ctx context.Context, jobID uuid.UUID) (*domain.Job, error) {
	// Fetch job by ID directly - the job was already claimed via RabbitMQ/Redis queue
	job, err := r.client.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %s: %w", jobID, err)
	}

	if job == nil {
		log.Printf("job %s not found (may have been cancelled)", jobID)
		return nil, nil
	}

	// Verify job is in a processable state
	if job.Status != domain.JobStatusPending && job.Status != domain.JobStatusRunning {
		log.Printf("job %s has status %s, skipping", jobID, job.Status)
		return nil, nil
	}

	return job, nil
}

func (r *Runner) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(domain.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			status := domain.WorkerStatusIdle
			var jobID *uuid.UUID

			if currentJob := r.getCurrentJob(); currentJob != nil {
				status = domain.WorkerStatusBusy
				jobID = &currentJob.ID
			}

			if err := r.client.Heartbeat(ctx, status, jobID); err != nil {
				log.Printf("warning: heartbeat failed: %v", err)
			}
		}
	}
}

func (r *Runner) workLoop(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.stopChan:
			return nil
		case <-ticker.C:
			// Try to claim a job
			job, err := r.client.ClaimJob(ctx)
			if err != nil {
				log.Printf("error claiming job: %v", err)
				continue
			}

			if job == nil {
				// No pending jobs
				continue
			}

			r.setCurrentJob(job)
			log.Printf("claimed job: %s (%s)", job.Name, job.ID)

			// Process the job
			placesScraped, err := r.processJob(ctx, job)
			if err != nil {
				log.Printf("job failed: %s - %v", job.ID, err)
				if failErr := r.client.FailJob(ctx, job.ID, err.Error()); failErr != nil {
					log.Printf("warning: failed to mark job as failed: %v", failErr)
				}
			} else {
				log.Printf("job completed: %s (%d places)", job.ID, placesScraped)
				if completeErr := r.client.CompleteJob(ctx, job.ID, placesScraped); completeErr != nil {
					log.Printf("warning: failed to mark job as completed: %v", completeErr)
				}
			}

			r.setCurrentJob(nil)
		}
	}
}

func (r *Runner) processJob(ctx context.Context, job *domain.Job) (int, error) {
	if len(job.Config.Keywords) == 0 {
		return 0, errors.New("no keywords provided")
	}

	outpath := filepath.Join(r.dataFolder, job.ID.String()+".csv")

	outfile, err := os.Create(outpath)
	if err != nil {
		return 0, err
	}
	defer outfile.Close()

	csvWriter := csvwriter.NewCsvWriter(csv.NewWriter(outfile))
	memWriter := &MemoryWriter{}
	writers := []scrapemate.ResultWriter{csvWriter, memWriter}

	mate, err := r.setupMate(ctx, writers, job)
	if err != nil {
		return 0, err
	}
	defer mate.Close()

	var coords string
	if job.Config.GeoLat != nil && job.Config.GeoLon != nil {
		coords = formatCoords(*job.Config.GeoLat, *job.Config.GeoLon)
	}

	// Use Redis deduper if available, otherwise use local in-memory deduper
	var dedup deduper.Deduper
	if r.redisDeduper != nil {
		dedup = r.redisDeduper
		log.Printf("job %s: using Redis distributed deduplication", job.ID)
	} else {
		dedup = deduper.New()
		log.Printf("job %s: using local in-memory deduplication", job.ID)
	}
	exitMonitor := exiter.New()

	var ev emailvalidator.Validator
	if r.config.EmailValidatorKey != "" {
		ev = emailvalidator.NewMoribouncerValidator(emailvalidator.Config{
			APIKey: r.config.EmailValidatorKey,
			APIURL: r.config.EmailValidatorURL,
		})
	}

	seedJobs, err := runner.CreateSeedJobs(
		job.Config.FastMode,
		job.Config.Lang,
		strings.NewReader(strings.Join(job.Config.Keywords, "\n")),
		job.Config.Depth,
		job.Config.ExtractEmail,
		coords,
		job.Config.Zoom,
		func() float64 {
			if job.Config.Radius <= 0 {
				return 10000
			}
			return float64(job.Config.Radius)
		}(),
		dedup,
		exitMonitor,
		ev,
		r.config.ExtraReviews,
	)
	if err != nil {
		return 0, err
	}

	if len(seedJobs) == 0 {
		return 0, nil
	}

	exitMonitor.SetSeedCount(len(seedJobs))

	allowedSeconds := max(60, len(seedJobs)*10*job.Config.Depth/50+120)

	if job.Config.MaxTime > 0 {
		if job.Config.MaxTime.Seconds() < 180 {
			allowedSeconds = 180
		} else {
			allowedSeconds = int(job.Config.MaxTime.Seconds())
		}
	}

	log.Printf("running job %s with %d seed jobs and %d allowed seconds",
		job.ID, len(seedJobs), allowedSeconds)

	mateCtx, cancel := context.WithTimeout(ctx, time.Duration(allowedSeconds)*time.Second)
	defer cancel()

	exitMonitor.SetCancelFunc(cancel)
	go exitMonitor.Run(mateCtx)

	err = mate.Start(mateCtx, seedJobs...)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		cancel()
		return 0, err
	}

	cancel()
	mate.Close()

	// Submit results to manager
	results := memWriter.GetResults()
	log.Printf("[Worker] Job %s: CSV written, MemoryWriter has %d results", job.ID, len(results))

	if len(results) > 0 {
		log.Printf("[Worker] Job %s: Submitting %d results to manager at %s", job.ID, len(results), r.client.baseURL)
		if err := r.client.SubmitResults(ctx, job.ID, results); err != nil {
			log.Printf("[Worker] Job %s: SubmitResults FAILED: %v", job.ID, err)
			return 0, fmt.Errorf("failed to submit results: %w", err)
		}
		log.Printf("[Worker] Job %s: Results submitted successfully", job.ID)
	} else {
		log.Printf("[Worker] Job %s: No results in MemoryWriter (check UseInResults)", job.ID)
	}

	return len(results), nil
}

func (r *Runner) setupMate(_ context.Context, writers []scrapemate.ResultWriter, job *domain.Job) (*scrapemateapp.ScrapemateApp, error) {
	opts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(r.config.Concurrency),
		scrapemateapp.WithExitOnInactivity(time.Minute * 3),
	}

	if !job.Config.FastMode {
		opts = append(opts,
			scrapemateapp.WithJS(scrapemateapp.DisableImages()),
		)
	} else {
		opts = append(opts,
			scrapemateapp.WithStealth("firefox"),
		)
	}

	hasProxy := false

	if len(r.config.Proxies) > 0 {
		opts = append(opts, scrapemateapp.WithProxies(r.config.Proxies))
		hasProxy = true
	} else if len(job.Config.Proxies) > 0 {
		opts = append(opts,
			scrapemateapp.WithProxies(job.Config.Proxies),
		)
		hasProxy = true
	}

	if !r.config.DisablePageReuse {
		opts = append(opts,
			scrapemateapp.WithPageReuseLimit(2),
			scrapemateapp.WithPageReuseLimit(200),
		)
	}

	log.Printf("job %s has proxy: %v", job.ID, hasProxy)

	matecfg, err := scrapemateapp.NewConfig(
		writers,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return scrapemateapp.NewScrapeMateApp(matecfg)
}

func formatCoords(lat, lon float64) string {
	return fmt.Sprintf("%f,%f", lat, lon)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
