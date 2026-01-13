package queue

import (
	"context"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, payload *JobPayload) error

// Worker processes jobs from the Redis queue
type Worker struct {
	server  *asynq.Server
	mux     *asynq.ServeMux
	handler JobHandler
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	RedisURL    string
	RedisAddr   string
	Password    string
	DB          int
	Concurrency int
	Queues      map[string]int // queue name -> priority
}

// NewWorker creates a new queue worker
func NewWorker(cfg *WorkerConfig, handler JobHandler) (*Worker, error) {
	var redisOpt asynq.RedisConnOpt

	if cfg.RedisURL != "" {
		opt, err := asynq.ParseRedisURI(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
		redisOpt = opt
	} else if cfg.RedisAddr != "" {
		redisOpt = asynq.RedisClientOpt{
			Addr:     cfg.RedisAddr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}
	} else {
		return nil, fmt.Errorf("redis URL or address is required")
	}

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	queues := cfg.Queues
	if queues == nil {
		// Default queue priorities
		queues = map[string]int{
			QueueCritical: 6,
			QueueHigh:     3,
			QueueDefault:  2,
			QueueLow:      1,
		}
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("queue worker error: task=%s, error=%v", task.Type(), err)
			}),
			Logger: &asynqLogger{},
		},
	)

	mux := asynq.NewServeMux()

	w := &Worker{
		server:  server,
		mux:     mux,
		handler: handler,
	}

	// Register the job handler
	mux.HandleFunc(TypeJobProcess, w.handleJob)

	return w, nil
}

// handleJob processes a job task
func (w *Worker) handleJob(ctx context.Context, task *asynq.Task) error {
	payload, err := ParsePayload(task.Payload())
	if err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	log.Printf("queue worker: processing job %s", payload.JobID)

	if err := w.handler(ctx, payload); err != nil {
		log.Printf("queue worker: job %s failed: %v", payload.JobID, err)
		return err
	}

	log.Printf("queue worker: job %s completed", payload.JobID)
	return nil
}

// Run starts the worker
func (w *Worker) Run(ctx context.Context) error {
	// Run server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- w.server.Run(w.mux)
	}()

	select {
	case <-ctx.Done():
		w.server.Shutdown()
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the worker
func (w *Worker) Shutdown() {
	if w.server != nil {
		w.server.Shutdown()
	}
}

// asynqLogger adapts asynq logging to standard log
type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...interface{}) {
	// Suppress debug logs
}

func (l *asynqLogger) Info(args ...interface{}) {
	log.Println(args...)
}

func (l *asynqLogger) Warn(args ...interface{}) {
	log.Println("[WARN]", fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...interface{}) {
	log.Println("[ERROR]", fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	log.Fatalln("[FATAL]", fmt.Sprint(args...))
}
