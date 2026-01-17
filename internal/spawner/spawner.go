// Package spawner provides interfaces and implementations for auto-spawning
// worker containers/functions when jobs are created.
//
// Supported spawners:
//   - Docker: Spawns local Docker containers
//   - Swarm: Spawns Docker Swarm services (for Dokploy)
//   - Lambda: Triggers AWS Lambda functions
package spawner

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SpawnRequest contains the information needed to spawn a worker
type SpawnRequest struct {
	// JobID is the UUID of the job to process
	JobID uuid.UUID

	// Priority of the job (higher = more urgent)
	Priority int

	// ManagerURL is the URL workers should connect to
	ManagerURL string

	// RabbitMQURL is the message queue URL
	RabbitMQURL string

	// RedisAddr is the Redis address for queue/cache
	RedisAddr string

	// Concurrency is the number of concurrent scrapers per worker
	Concurrency int

	// WorkerImage is the Docker image to use (for Docker/Swarm spawners)
	WorkerImage string

	// ExtraArgs are additional arguments to pass to the worker
	ExtraArgs []string
}

// SpawnResult contains the result of a spawn operation
type SpawnResult struct {
	// WorkerID is the identifier of the spawned worker (container ID, task ARN, etc.)
	WorkerID string

	// Status is the current status of the spawn operation
	Status string

	// Error message if spawn failed
	Error string
}

// Spawner is the interface for spawning workers on-demand
type Spawner interface {
	// Spawn starts a new worker to process a job
	Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error)

	// Status checks the status of a spawned worker
	Status(ctx context.Context, workerID string) (*SpawnResult, error)

	// Stop terminates a spawned worker
	Stop(ctx context.Context, workerID string) error

	// Close cleans up spawner resources
	Close() error

	// Name returns the spawner type name
	Name() string
}

// SpawnerType represents the type of spawner
type SpawnerType string

const (
	SpawnerTypeNone   SpawnerType = "none"
	SpawnerTypeDocker SpawnerType = "docker"
	SpawnerTypeSwarm  SpawnerType = "swarm"
	SpawnerTypeLambda SpawnerType = "lambda"
)

// Config holds configuration for spawner initialization
type Config struct {
	// Type is the spawner type to use
	Type SpawnerType

	// ManagerURL is the URL workers should connect to
	ManagerURL string

	// RabbitMQURL is the message queue URL
	RabbitMQURL string

	// RedisAddr is the Redis address
	RedisAddr string

	// Proxies is the proxy URL for workers (e.g., "socks5://host:port")
	Proxies string

	// Docker-specific configuration
	Docker DockerConfig

	// Swarm-specific configuration (Dokploy)
	Swarm SwarmConfig

	// Lambda-specific configuration
	Lambda LambdaConfig
}

// DockerConfig holds Docker spawner configuration
type DockerConfig struct {
	// Image is the Docker image to use for workers
	Image string

	// Network is the Docker network to attach workers to
	Network string

	// Concurrency is the default concurrency per worker
	Concurrency int

	// AutoRemove removes containers after they exit
	AutoRemove bool

	// MaxWorkers is the maximum number of concurrent workers (0 = unlimited)
	MaxWorkers int

	// Environment variables to pass to workers
	Environment map[string]string
}

// SwarmConfig holds Docker Swarm spawner configuration (for Dokploy)
type SwarmConfig struct {
	// Image is the Docker image to use for workers
	Image string

	// Network is the overlay network to attach workers to
	Network string

	// Concurrency is the default concurrency per worker
	Concurrency int

	// Replicas is the number of replicas per service
	Replicas int

	// MaxServices is the maximum number of concurrent services
	MaxServices int

	// Labels to apply to spawned services
	Labels map[string]string

	// Constraints for service placement
	Constraints []string

	// Environment variables to pass to workers
	Environment map[string]string
}

// LambdaConfig holds AWS Lambda spawner configuration
type LambdaConfig struct {
	// FunctionName is the Lambda function name or ARN
	FunctionName string

	// Region is the AWS region
	Region string

	// InvocationType is "Event" (async) or "RequestResponse" (sync)
	InvocationType string

	// MaxConcurrent is the maximum concurrent Lambda invocations
	MaxConcurrent int
}

// New creates a new spawner based on configuration
func New(cfg *Config) (Spawner, error) {
	switch cfg.Type {
	case SpawnerTypeNone, "":
		return NewNoOpSpawner(), nil
	case SpawnerTypeDocker:
		return NewDockerSpawner(&cfg.Docker, cfg.ManagerURL, cfg.RabbitMQURL, cfg.RedisAddr, cfg.Proxies)
	case SpawnerTypeSwarm:
		return NewSwarmSpawner(&cfg.Swarm, cfg.ManagerURL, cfg.RabbitMQURL, cfg.RedisAddr)
	case SpawnerTypeLambda:
		return NewLambdaSpawner(&cfg.Lambda, cfg.ManagerURL, cfg.RabbitMQURL, cfg.RedisAddr)
	default:
		return nil, fmt.Errorf("unknown spawner type: %s", cfg.Type)
	}
}

// NoOpSpawner is a spawner that does nothing (for when auto-spawn is disabled)
type NoOpSpawner struct{}

// NewNoOpSpawner creates a new no-op spawner
func NewNoOpSpawner() *NoOpSpawner {
	return &NoOpSpawner{}
}

func (s *NoOpSpawner) Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error) {
	return &SpawnResult{
		WorkerID: "",
		Status:   "disabled",
	}, nil
}

func (s *NoOpSpawner) Status(ctx context.Context, workerID string) (*SpawnResult, error) {
	return &SpawnResult{Status: "unknown"}, nil
}

func (s *NoOpSpawner) Stop(ctx context.Context, workerID string) error {
	return nil
}

func (s *NoOpSpawner) Close() error {
	return nil
}

func (s *NoOpSpawner) Name() string {
	return "none"
}
