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
	"time"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/deduper"
	"github.com/sadewadee/google-scraper/exiter"
	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/runner"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/adapters/writers/csvwriter"
	"github.com/gosom/scrapemate/scrapemateapp"
)

// Runner is a worker that claims and processes jobs from the manager
type Runner struct {
	client      *Client
	config      *runner.Config
	dataFolder  string
	currentJob  *domain.Job
	stopChan    chan struct{}
	workerID    string
}

// NewRunner creates a new worker runner
func NewRunner(managerURL, workerID string, cfg *runner.Config) (*Runner, error) {
	if cfg.DataFolder == "" {
		cfg.DataFolder = "."
	}

	if err := os.MkdirAll(cfg.DataFolder, os.ModePerm); err != nil {
		return nil, err
	}

	return &Runner{
		client:     NewClient(managerURL, workerID),
		config:     cfg,
		dataFolder: cfg.DataFolder,
		workerID:   workerID,
		stopChan:   make(chan struct{}),
	}, nil
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

	// Main work loop
	return r.workLoop(ctx)
}

// Stop gracefully stops the worker
func (r *Runner) Stop(ctx context.Context) error {
	close(r.stopChan)

	// Release current job if any
	if r.currentJob != nil {
		if err := r.client.ReleaseJob(ctx, r.currentJob.ID); err != nil {
			log.Printf("warning: failed to release job: %v", err)
		}
	}

	// Unregister from manager
	if err := r.client.Unregister(ctx); err != nil {
		log.Printf("warning: failed to unregister: %v", err)
	}

	return nil
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

			if r.currentJob != nil {
				status = domain.WorkerStatusBusy
				jobID = &r.currentJob.ID
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

			r.currentJob = job
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

			r.currentJob = nil
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

	dedup := deduper.New()
	exitMonitor := exiter.New()

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
