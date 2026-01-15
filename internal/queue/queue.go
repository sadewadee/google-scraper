// Package queue provides a Redis-based job queue using Asynq
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	// Task types
	TypeJobProcess = "job:process"

	// Queue names for regional affinity
	QueueDefault  = "default"
	QueueHigh     = "high"
	QueueLow      = "low"
	QueueCritical = "critical"
)

// JobPayload is the payload for a job processing task
type JobPayload struct {
	JobID     uuid.UUID `json:"job_id"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// Config holds Redis queue configuration
type Config struct {
	RedisURL  string
	RedisAddr string
	Password  string
	DB        int
}

// Queue is a Redis-based job queue
type Queue struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	redisOpt  asynq.RedisConnOpt
}

// New creates a new Queue
func New(cfg *Config) (*Queue, error) {
	var redisOpt asynq.RedisConnOpt

	if cfg.RedisURL != "" {
		opt, err := asynq.ParseRedisURI(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
		redisOpt = opt
	} else if cfg.RedisAddr != "" {
		redisOpt = asynq.RedisClientOpt{
			Addr:         cfg.RedisAddr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
		}
	} else {
		return nil, fmt.Errorf("redis URL or address is required")
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	return &Queue{
		client:    client,
		inspector: inspector,
		redisOpt:  redisOpt,
	}, nil
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(ctx context.Context, jobID uuid.UUID, priority int) error {
	payload := JobPayload{
		JobID:     jobID,
		Priority:  priority,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeJobProcess, data)

	// Set queue based on priority
	queueName := QueueDefault
	if priority >= 10 {
		queueName = QueueCritical
	} else if priority >= 5 {
		queueName = QueueHigh
	} else if priority < 0 {
		queueName = QueueLow
	}

	opts := []asynq.Option{
		asynq.Queue(queueName),
		asynq.MaxRetry(3),
		asynq.Timeout(30 * time.Minute),
		asynq.Retention(24 * time.Hour),
	}

	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Printf("queue: enqueued job %s to queue %s (task_id: %s)", jobID, queueName, info.ID)
	return nil
}

// GetRedisOpt returns the Redis client options for creating a server
func (q *Queue) GetRedisOpt() asynq.RedisConnOpt {
	return q.redisOpt
}

// GetQueueStats returns queue statistics
func (q *Queue) GetQueueStats(ctx context.Context) (map[string]*asynq.QueueInfo, error) {
	queues := []string{QueueDefault, QueueHigh, QueueLow, QueueCritical}
	stats := make(map[string]*asynq.QueueInfo)

	for _, queue := range queues {
		info, err := q.inspector.GetQueueInfo(queue)
		if err != nil {
			// Queue might not exist yet
			continue
		}
		stats[queue] = info
	}

	return stats, nil
}

// Close closes the queue client
func (q *Queue) Close() error {
	if q.client != nil {
		return q.client.Close()
	}
	return nil
}

// ParsePayload parses a job payload from task data
func ParsePayload(data []byte) (*JobPayload, error) {
	var payload JobPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return &payload, nil
}
