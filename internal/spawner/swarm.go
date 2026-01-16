package spawner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// SwarmSpawner spawns workers as Docker Swarm services
// This is ideal for Dokploy deployments where Swarm manages the cluster
type SwarmSpawner struct {
	client      *client.Client
	cfg         *SwarmConfig
	managerURL  string
	rabbitmqURL string
	redisAddr   string

	// Track active services
	mu       sync.Mutex
	services map[string]time.Time
}

// NewSwarmSpawner creates a new Docker Swarm spawner
func NewSwarmSpawner(cfg *SwarmConfig, managerURL, rabbitmqURL, redisAddr string) (*SwarmSpawner, error) {
	// Set defaults
	if cfg.Image == "" {
		cfg.Image = "gmaps-scraper:latest"
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}
	if cfg.Replicas == 0 {
		cfg.Replicas = 1
	}
	if cfg.Network == "" {
		cfg.Network = "gmaps-network"
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection and verify Swarm mode
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := cli.Info(ctx)
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}

	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		cli.Close()
		return nil, fmt.Errorf("Docker Swarm is not active (state: %s)", info.Swarm.LocalNodeState)
	}

	log.Printf("[SwarmSpawner] Connected to Docker Swarm, image=%s, network=%s", cfg.Image, cfg.Network)

	return &SwarmSpawner{
		client:      cli,
		cfg:         cfg,
		managerURL:  managerURL,
		rabbitmqURL: rabbitmqURL,
		redisAddr:   redisAddr,
		services:    make(map[string]time.Time),
	}, nil
}

func (s *SwarmSpawner) Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error) {
	// Check max services limit and reserve a slot atomically
	reservationKey := "pending-" + req.JobID.String()
	s.mu.Lock()
	activeCount := len(s.services)
	if s.cfg.MaxServices > 0 && activeCount >= s.cfg.MaxServices {
		s.mu.Unlock()
		log.Printf("[SwarmSpawner] Max services reached (%d/%d), skipping spawn for job %s",
			activeCount, s.cfg.MaxServices, req.JobID)
		return &SpawnResult{
			Status: "skipped",
			Error:  "max services limit reached",
		}, nil
	}
	// Reserve a slot to prevent race condition
	s.services[reservationKey] = time.Now()
	s.mu.Unlock()

	// Ensure reservation is cleaned up on failure
	defer func() {
		s.mu.Lock()
		delete(s.services, reservationKey)
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

	// Build environment variables as strings
	envStrings := []string{
		fmt.Sprintf("JOB_ID=%s", req.JobID.String()),
		fmt.Sprintf("JOB_PRIORITY=%d", req.Priority),
	}
	for k, v := range s.cfg.Environment {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", k, v))
	}

	// Service name
	serviceName := fmt.Sprintf("gmaps-worker-%s", req.JobID.String()[:8])

	// Build labels
	labels := map[string]string{
		"gmaps.worker":   "true",
		"gmaps.job-id":   req.JobID.String(),
		"gmaps.priority": fmt.Sprintf("%d", req.Priority),
		"gmaps.spawner":  "swarm",
	}
	for k, v := range s.cfg.Labels {
		labels[k] = v
	}

	// Build placement constraints
	var placement *swarm.Placement
	if len(s.cfg.Constraints) > 0 {
		placement = &swarm.Placement{
			Constraints: s.cfg.Constraints,
		}
	}

	// Build network attachments
	networks := []swarm.NetworkAttachmentConfig{
		{Target: s.cfg.Network},
	}

	// Replicas
	replicas := uint64(s.cfg.Replicas)

	// Service spec
	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   serviceName,
			Labels: labels,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:   s.cfg.Image,
				Command: cmd,
				Env:     envStrings,
			},
			Placement: placement,
			Networks:  networks,
			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionNone, // Don't restart, worker should exit after job
			},
		},
		Mode: swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{
				Replicas: &replicas,
			},
		},
	}

	// Create service
	log.Printf("[SwarmSpawner] Creating service %s for job %s", serviceName, req.JobID)

	resp, err := s.client.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Track service
	s.mu.Lock()
	s.services[resp.ID] = time.Now()
	s.mu.Unlock()

	log.Printf("[SwarmSpawner] Created service %s (ID: %s) for job %s",
		serviceName, resp.ID[:12], req.JobID)

	return &SpawnResult{
		WorkerID: resp.ID,
		Status:   "running",
	}, nil
}

func (s *SwarmSpawner) Status(ctx context.Context, workerID string) (*SpawnResult, error) {
	service, _, err := s.client.ServiceInspectWithRaw(ctx, workerID, types.ServiceInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect service: %w", err)
	}

	// Get task status
	tasks, err := s.client.TaskList(ctx, types.TaskListOptions{
		Filters: filtersFromMap(map[string][]string{
			"service": {service.ID},
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	status := "unknown"
	if len(tasks) > 0 {
		// Get the most recent task
		latestTask := tasks[0]
		for _, task := range tasks {
			if task.CreatedAt.After(latestTask.CreatedAt) {
				latestTask = task
			}
		}

		switch latestTask.Status.State {
		case swarm.TaskStateRunning:
			status = "running"
		case swarm.TaskStateComplete:
			status = "completed"
		case swarm.TaskStateFailed:
			status = "failed"
		case swarm.TaskStatePending, swarm.TaskStateAssigned, swarm.TaskStateAccepted, swarm.TaskStatePreparing, swarm.TaskStateReady, swarm.TaskStateStarting:
			status = "starting"
		case swarm.TaskStateShutdown:
			status = "shutdown"
		case swarm.TaskStateRejected:
			status = "rejected"
		default:
			status = string(latestTask.Status.State)
		}
	}

	return &SpawnResult{
		WorkerID: workerID,
		Status:   status,
	}, nil
}

func (s *SwarmSpawner) Stop(ctx context.Context, workerID string) error {
	if err := s.client.ServiceRemove(ctx, workerID); err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}

	s.mu.Lock()
	delete(s.services, workerID)
	s.mu.Unlock()

	log.Printf("[SwarmSpawner] Removed service %s", workerID[:12])
	return nil
}

func (s *SwarmSpawner) Close() error {
	return s.client.Close()
}

func (s *SwarmSpawner) Name() string {
	return "swarm"
}

// CleanupCompleted removes services that have completed
func (s *SwarmSpawner) CleanupCompleted(ctx context.Context) error {
	s.mu.Lock()
	serviceIDs := make([]string, 0, len(s.services))
	for id := range s.services {
		serviceIDs = append(serviceIDs, id)
	}
	s.mu.Unlock()

	for _, id := range serviceIDs {
		result, err := s.Status(ctx, id)
		if err != nil {
			// Service doesn't exist anymore
			s.mu.Lock()
			delete(s.services, id)
			s.mu.Unlock()
			continue
		}

		if result.Status == "completed" || result.Status == "failed" {
			// Remove completed/failed service
			if err := s.Stop(ctx, id); err != nil {
				log.Printf("[SwarmSpawner] Warning: failed to remove completed service %s: %v", id[:12], err)
			}
		}
	}

	return nil
}

// ActiveCount returns the number of active services
func (s *SwarmSpawner) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.services)
}

// filtersFromMap converts a map to Docker filter args
func filtersFromMap(m map[string][]string) filters.Args {
	args := filters.NewArgs()
	for k, vals := range m {
		for _, v := range vals {
			args.Add(k, v)
		}
	}
	return args
}
