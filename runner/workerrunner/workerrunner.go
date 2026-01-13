package workerrunner

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/worker"
	"github.com/sadewadee/google-scraper/runner"
)

// Config holds configuration for the worker runner
type Config struct {
	// ManagerURL is the manager API URL
	ManagerURL string

	// WorkerID is the unique worker identifier (auto-generated if empty)
	WorkerID string

	// RunnerConfig is the scraping configuration
	RunnerConfig *runner.Config

	// Redis configuration for job queue
	RedisURL  string
	RedisAddr string
	RedisPass string
	RedisDB   int
}

// WorkerRunner runs a worker that claims and processes jobs
type WorkerRunner struct {
	cfg    *Config
	runner *worker.Runner
}

// New creates a new WorkerRunner
func New(cfg *Config) (runner.Runner, error) {
	if cfg.ManagerURL == "" {
		return nil, fmt.Errorf("manager URL is required")
	}

	if cfg.WorkerID == "" {
		hostname, _ := os.Hostname()
		cfg.WorkerID = fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
	}

	if cfg.RunnerConfig == nil {
		cfg.RunnerConfig = &runner.Config{}
	}

	if cfg.RunnerConfig.DataFolder == "" {
		cfg.RunnerConfig.DataFolder = "."
	}

	workerCfg := &worker.Config{
		ManagerURL:   cfg.ManagerURL,
		WorkerID:     cfg.WorkerID,
		RunnerConfig: cfg.RunnerConfig,
		RedisURL:     cfg.RedisURL,
		RedisAddr:    cfg.RedisAddr,
		RedisPass:    cfg.RedisPass,
		RedisDB:      cfg.RedisDB,
	}

	r, err := worker.NewRunner(workerCfg)
	if err != nil {
		return nil, err
	}

	return &WorkerRunner{
		cfg:    cfg,
		runner: r,
	}, nil
}

// Run starts the worker
func (w *WorkerRunner) Run(ctx context.Context) error {
	log.Printf("starting worker %s connecting to %s", w.cfg.WorkerID, w.cfg.ManagerURL)

	return w.runner.Run(ctx)
}

// Close cleans up resources
func (w *WorkerRunner) Close(ctx context.Context) error {
	if w.runner != nil {
		return w.runner.Stop(ctx)
	}
	return nil
}
