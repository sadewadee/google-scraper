package spawner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// DockerSpawner spawns workers as local Docker containers
type DockerSpawner struct {
	client      *client.Client
	cfg         *DockerConfig
	managerURL  string
	rabbitmqURL string
	redisAddr   string

	// Track active containers
	mu         sync.Mutex
	containers map[string]time.Time
}

// NewDockerSpawner creates a new Docker spawner
func NewDockerSpawner(cfg *DockerConfig, managerURL, rabbitmqURL, redisAddr string) (*DockerSpawner, error) {
	// Set defaults
	if cfg.Image == "" {
		cfg.Image = "gmaps-scraper:latest"
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}
	if cfg.Network == "" {
		cfg.Network = "gmaps-network"
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	log.Printf("[DockerSpawner] Connected to Docker, image=%s, network=%s", cfg.Image, cfg.Network)

	return &DockerSpawner{
		client:      cli,
		cfg:         cfg,
		managerURL:  managerURL,
		rabbitmqURL: rabbitmqURL,
		redisAddr:   redisAddr,
		containers:  make(map[string]time.Time),
	}, nil
}

func (s *DockerSpawner) Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error) {
	// Check max workers limit and reserve a slot atomically
	reservationKey := "pending-" + req.JobID.String()
	s.mu.Lock()
	activeCount := len(s.containers)
	if s.cfg.MaxWorkers > 0 && activeCount >= s.cfg.MaxWorkers {
		s.mu.Unlock()
		log.Printf("[DockerSpawner] Max workers reached (%d/%d), skipping spawn for job %s",
			activeCount, s.cfg.MaxWorkers, req.JobID)
		return &SpawnResult{
			Status: "skipped",
			Error:  "max workers limit reached",
		}, nil
	}
	// Reserve a slot to prevent race condition
	s.containers[reservationKey] = time.Now()
	s.mu.Unlock()

	// Ensure reservation is cleaned up on failure
	defer func() {
		s.mu.Lock()
		delete(s.containers, reservationKey)
		s.mu.Unlock()
	}()

	// Build container command
	cmd := []string{
		"-worker",
		"-manager-url", s.managerURL,
		"-c", fmt.Sprintf("%d", s.cfg.Concurrency),
	}

	if s.rabbitmqURL != "" {
		cmd = append(cmd, "-rabbitmq-url", s.rabbitmqURL)
	}
	if s.redisAddr != "" {
		cmd = append(cmd, "-redis-addr", s.redisAddr)
	}

	// Add extra args
	cmd = append(cmd, req.ExtraArgs...)

	// Build environment variables
	env := []string{
		fmt.Sprintf("JOB_ID=%s", req.JobID),
		fmt.Sprintf("JOB_PRIORITY=%d", req.Priority),
	}
	for k, v := range s.cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Container configuration
	containerName := fmt.Sprintf("gmaps-worker-%s", req.JobID.String()[:8])

	containerCfg := &container.Config{
		Image: s.cfg.Image,
		Cmd:   cmd,
		Env:   env,
		Labels: map[string]string{
			"gmaps.worker":   "true",
			"gmaps.job-id":   req.JobID.String(),
			"gmaps.priority": fmt.Sprintf("%d", req.Priority),
			"gmaps.spawner":  "docker",
		},
	}

	hostCfg := &container.HostConfig{
		AutoRemove:  s.cfg.AutoRemove,
		NetworkMode: container.NetworkMode(s.cfg.Network),
	}

	networkCfg := &network.NetworkingConfig{}

	// Create container
	log.Printf("[DockerSpawner] Creating container %s for job %s", containerName, req.JobID)

	resp, err := s.client.ContainerCreate(ctx, containerCfg, hostCfg, networkCfg, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := s.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up created container
		s.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Track container
	s.mu.Lock()
	s.containers[resp.ID] = time.Now()
	s.mu.Unlock()

	log.Printf("[DockerSpawner] Started container %s (ID: %s) for job %s",
		containerName, resp.ID[:12], req.JobID)

	return &SpawnResult{
		WorkerID: resp.ID,
		Status:   "running",
	}, nil
}

func (s *DockerSpawner) Status(ctx context.Context, workerID string) (*SpawnResult, error) {
	inspect, err := s.client.ContainerInspect(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	status := "unknown"
	switch {
	case inspect.State.Running:
		status = "running"
	case inspect.State.Paused:
		status = "paused"
	case inspect.State.Restarting:
		status = "restarting"
	case inspect.State.Dead:
		status = "dead"
	case inspect.State.ExitCode == 0:
		status = "completed"
	default:
		status = "failed"
	}

	return &SpawnResult{
		WorkerID: workerID,
		Status:   status,
	}, nil
}

func (s *DockerSpawner) Stop(ctx context.Context, workerID string) error {
	timeout := 30 // seconds

	if err := s.client.ContainerStop(ctx, workerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	s.mu.Lock()
	delete(s.containers, workerID)
	s.mu.Unlock()

	log.Printf("[DockerSpawner] Stopped container %s", workerID[:12])
	return nil
}

func (s *DockerSpawner) Close() error {
	return s.client.Close()
}

func (s *DockerSpawner) Name() string {
	return "docker"
}

// CleanupCompleted removes tracking for containers that have exited
func (s *DockerSpawner) CleanupCompleted(ctx context.Context) error {
	s.mu.Lock()
	containerIDs := make([]string, 0, len(s.containers))
	for id := range s.containers {
		containerIDs = append(containerIDs, id)
	}
	s.mu.Unlock()

	for _, id := range containerIDs {
		inspect, err := s.client.ContainerInspect(ctx, id)
		if err != nil {
			// Container doesn't exist anymore
			s.mu.Lock()
			delete(s.containers, id)
			s.mu.Unlock()
			continue
		}

		if !inspect.State.Running {
			s.mu.Lock()
			delete(s.containers, id)
			s.mu.Unlock()
		}
	}

	return nil
}

// ActiveCount returns the number of active containers
func (s *DockerSpawner) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.containers)
}
