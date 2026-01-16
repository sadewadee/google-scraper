package spawner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// LambdaSpawner spawns workers by invoking AWS Lambda functions
type LambdaSpawner struct {
	client      *lambda.Client
	cfg         *LambdaConfig
	managerURL  string
	rabbitmqURL string
	redisAddr   string

	// Track active invocations
	mu          sync.Mutex
	invocations map[string]invocationInfo // requestID -> info
	activeCount int64
}

// invocationInfo tracks invocation state
type invocationInfo struct {
	status  string
	isAsync bool // Only async invocations count toward activeCount
}

// LambdaPayload is the payload sent to the Lambda function
type LambdaPayload struct {
	JobID       string `json:"job_id"`
	Priority    int    `json:"priority"`
	ManagerURL  string `json:"manager_url"`
	RabbitMQURL string `json:"rabbitmq_url,omitempty"`
	RedisAddr   string `json:"redis_addr,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

// NewLambdaSpawner creates a new AWS Lambda spawner
func NewLambdaSpawner(cfg *LambdaConfig, managerURL, rabbitmqURL, redisAddr string) (*LambdaSpawner, error) {
	// Set defaults
	if cfg.InvocationType == "" {
		cfg.InvocationType = "Event" // Async by default
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 100 // Lambda default concurrent limit
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Lambda client
	client := lambda.NewFromConfig(awsCfg)

	// Verify function exists
	_, err = client.GetFunction(context.Background(), &lambda.GetFunctionInput{
		FunctionName: aws.String(cfg.FunctionName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Lambda function %s: %w", cfg.FunctionName, err)
	}

	log.Printf("[LambdaSpawner] Connected to AWS Lambda, function=%s, region=%s", cfg.FunctionName, cfg.Region)

	return &LambdaSpawner{
		client:      client,
		cfg:         cfg,
		managerURL:  managerURL,
		rabbitmqURL: rabbitmqURL,
		redisAddr:   redisAddr,
		invocations: make(map[string]invocationInfo),
	}, nil
}

func (s *LambdaSpawner) Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error) {
	// Check concurrency limit
	current := atomic.LoadInt64(&s.activeCount)
	if s.cfg.MaxConcurrent > 0 && int(current) >= s.cfg.MaxConcurrent {
		log.Printf("[LambdaSpawner] Max concurrent invocations reached (%d/%d), skipping spawn for job %s",
			current, s.cfg.MaxConcurrent, req.JobID)
		return &SpawnResult{
			Status: "skipped",
			Error:  "max concurrent limit reached",
		}, nil
	}

	// Build payload
	payload := LambdaPayload{
		JobID:       req.JobID.String(),
		Priority:    req.Priority,
		ManagerURL:  s.managerURL,
		RabbitMQURL: s.rabbitmqURL,
		RedisAddr:   s.redisAddr,
		Concurrency: req.Concurrency,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Determine invocation type
	var invocationType lambdatypes.InvocationType
	switch s.cfg.InvocationType {
	case "Event":
		invocationType = lambdatypes.InvocationTypeEvent
	case "RequestResponse":
		invocationType = lambdatypes.InvocationTypeRequestResponse
	default:
		invocationType = lambdatypes.InvocationTypeEvent
	}

	log.Printf("[LambdaSpawner] Invoking Lambda function %s for job %s", s.cfg.FunctionName, req.JobID)

	// Invoke Lambda
	result, err := s.client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(s.cfg.FunctionName),
		Payload:        payloadBytes,
		InvocationType: invocationType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	// Track invocation
	requestID := ""
	if result.FunctionError != nil {
		return &SpawnResult{
			Status: "failed",
			Error:  *result.FunctionError,
		}, nil
	}

	// For async invocations, we get the request ID from the response
	// For sync invocations, we get the full response
	if invocationType == lambdatypes.InvocationTypeEvent {
		// Async - Lambda accepted the request
		requestID = req.JobID.String() // Use job ID as tracking ID
		atomic.AddInt64(&s.activeCount, 1)

		s.mu.Lock()
		s.invocations[requestID] = invocationInfo{status: "invoked", isAsync: true}
		s.mu.Unlock()
	} else {
		// Sync - Lambda completed, don't count toward active
		requestID = req.JobID.String()
		s.mu.Lock()
		s.invocations[requestID] = invocationInfo{status: "completed", isAsync: false}
		s.mu.Unlock()
	}

	log.Printf("[LambdaSpawner] Lambda invoked for job %s (status: %d)", req.JobID, result.StatusCode)

	return &SpawnResult{
		WorkerID: requestID,
		Status:   "invoked",
	}, nil
}

func (s *LambdaSpawner) Status(ctx context.Context, workerID string) (*SpawnResult, error) {
	s.mu.Lock()
	info, ok := s.invocations[workerID]
	s.mu.Unlock()

	if !ok {
		return &SpawnResult{
			WorkerID: workerID,
			Status:   "unknown",
		}, nil
	}

	return &SpawnResult{
		WorkerID: workerID,
		Status:   info.status,
	}, nil
}

func (s *LambdaSpawner) Stop(ctx context.Context, workerID string) error {
	// Lambda invocations cannot be stopped once started
	// We can only mark them as cancelled in our tracking
	s.mu.Lock()
	if info, ok := s.invocations[workerID]; ok {
		// Only decrement activeCount if this was an async invocation
		if info.isAsync && info.status != "completed" && info.status != "failed" && info.status != "cancelled" {
			atomic.AddInt64(&s.activeCount, -1)
		}
		s.invocations[workerID] = invocationInfo{status: "cancelled", isAsync: info.isAsync}
	}
	s.mu.Unlock()

	log.Printf("[LambdaSpawner] Marked invocation %s as cancelled (Lambda functions cannot be stopped)", workerID)
	return nil
}

func (s *LambdaSpawner) Close() error {
	// Nothing to close for Lambda client
	return nil
}

func (s *LambdaSpawner) Name() string {
	return "lambda"
}

// MarkCompleted marks an invocation as completed
// This should be called when the worker reports job completion
func (s *LambdaSpawner) MarkCompleted(workerID string) {
	s.mu.Lock()
	if info, ok := s.invocations[workerID]; ok {
		// Only decrement activeCount if this was an async invocation that's still active
		if info.isAsync && info.status != "completed" && info.status != "failed" && info.status != "cancelled" {
			atomic.AddInt64(&s.activeCount, -1)
		}
		s.invocations[workerID] = invocationInfo{status: "completed", isAsync: info.isAsync}
	}
	s.mu.Unlock()
}

// MarkFailed marks an invocation as failed
func (s *LambdaSpawner) MarkFailed(workerID string) {
	s.mu.Lock()
	if info, ok := s.invocations[workerID]; ok {
		// Only decrement activeCount if this was an async invocation that's still active
		if info.isAsync && info.status != "completed" && info.status != "failed" && info.status != "cancelled" {
			atomic.AddInt64(&s.activeCount, -1)
		}
		s.invocations[workerID] = invocationInfo{status: "failed", isAsync: info.isAsync}
	}
	s.mu.Unlock()
}

// ActiveCount returns the number of active invocations
func (s *LambdaSpawner) ActiveCount() int {
	return int(atomic.LoadInt64(&s.activeCount))
}

// CleanupOld removes old invocation tracking entries
func (s *LambdaSpawner) CleanupOld() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, info := range s.invocations {
		if info.status == "completed" || info.status == "failed" || info.status == "cancelled" {
			delete(s.invocations, id)
		}
	}
}
