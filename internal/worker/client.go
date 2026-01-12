package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// Client is a worker client that communicates with the manager API
type Client struct {
	baseURL    string
	workerID   string
	hostname   string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new worker client
func NewClient(baseURL, workerID string) *Client {
	hostname, _ := os.Hostname()
	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		apiToken = os.Getenv("API_KEY")
	}

	return &Client{
		baseURL:  baseURL,
		workerID: workerID,
		hostname: hostname,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Register registers the worker with the manager
func (c *Client) Register(ctx context.Context) (*domain.Worker, error) {
	body := map[string]string{
		"worker_id": c.workerID,
	}

	resp, err := c.post(ctx, "/api/v2/workers/register", body)
	if err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var worker domain.Worker
	if err := json.NewDecoder(resp.Body).Decode(&worker); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &worker, nil
}

// Heartbeat sends a heartbeat to the manager
func (c *Client) Heartbeat(ctx context.Context, status domain.WorkerStatus, currentJobID *uuid.UUID) error {
	body := map[string]interface{}{
		"worker_id": c.workerID,
		"hostname":  c.hostname,
		"status":    status,
	}

	if currentJobID != nil {
		body["current_job_id"] = currentJobID.String()
	}

	resp, err := c.post(ctx, "/api/v2/workers/heartbeat", body)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// ClaimJob claims a pending job from the manager
func (c *Client) ClaimJob(ctx context.Context) (*domain.Job, error) {
	url := fmt.Sprintf("/api/v2/workers/%s/claim", c.workerID)

	resp, err := c.post(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Job *domain.Job `json:"job"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Job, nil
}

// CompleteJob marks a job as completed
func (c *Client) CompleteJob(ctx context.Context, jobID uuid.UUID, placesScraped int) error {
	url := fmt.Sprintf("/api/v2/workers/%s/complete", c.workerID)

	body := map[string]interface{}{
		"job_id":         jobID.String(),
		"places_scraped": placesScraped,
	}

	resp, err := c.post(ctx, url, body)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// FailJob marks a job as failed
func (c *Client) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	url := fmt.Sprintf("/api/v2/workers/%s/fail", c.workerID)

	body := map[string]interface{}{
		"job_id":  jobID.String(),
		"message": errMsg,
	}

	resp, err := c.post(ctx, url, body)
	if err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// ReleaseJob releases a job back to pending
func (c *Client) ReleaseJob(ctx context.Context, jobID uuid.UUID) error {
	url := fmt.Sprintf("/api/v2/workers/%s/release", c.workerID)

	body := map[string]interface{}{
		"job_id": jobID.String(),
	}

	resp, err := c.post(ctx, url, body)
	if err != nil {
		return fmt.Errorf("failed to release job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// Unregister unregisters the worker from the manager
func (c *Client) Unregister(ctx context.Context) error {
	url := fmt.Sprintf("/api/v2/workers/%s", c.workerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// SubmitResults submits results to the manager
func (c *Client) SubmitResults(ctx context.Context, jobID uuid.UUID, data [][]byte) error {
	batch := domain.ResultBatch{
		JobID: jobID,
		Data:  data,
	}

	url := fmt.Sprintf("/api/v2/jobs/%s/results", jobID.String())
	resp, err := c.post(ctx, url, batch)
	if err != nil {
		return fmt.Errorf("failed to submit results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	return c.httpClient.Do(req)
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("request failed: %s", apiErr.Message)
}
