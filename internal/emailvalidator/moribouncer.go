package emailvalidator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// MoribouncerValidator validates emails using Moribouncer API
type MoribouncerValidator struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
}

// Config for Moribouncer validator
type Config struct {
	APIURL  string        // e.g., "https://api.moribouncer.com/v1"
	APIKey  string
	Timeout time.Duration
}

// NewMoribouncerValidator creates a new Moribouncer validator
func NewMoribouncerValidator(cfg Config) *MoribouncerValidator {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.moribouncer.com/v1"
	}

	return &MoribouncerValidator{
		apiURL: apiURL,
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// moribouncerResponse is the API response format from Moribouncer
type moribouncerResponse struct {
	Email       string  `json:"email"`
	Status      string  `json:"status"`
	Score       float64 `json:"score"`
	Deliverable bool    `json:"deliverable"`
	Disposable  bool    `json:"disposable"`
	RoleAccount bool    `json:"role_account"`
	FreeEmail   bool    `json:"free_email"`
	CatchAll    bool    `json:"catch_all"`
	Reason      string  `json:"reason"`
}

// Validate validates a single email
func (v *MoribouncerValidator) Validate(ctx context.Context, email string) (*ValidationResult, error) {
	// Build request URL
	reqURL := fmt.Sprintf("%s/validate?api_key=%s&email=%s",
		v.apiURL,
		url.QueryEscape(v.apiKey),
		url.QueryEscape(email),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Moribouncer API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Moribouncer API returned status %d", resp.StatusCode)
	}

	var apiResp moribouncerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ValidationResult{
		Email:       apiResp.Email,
		Status:      apiResp.Status,
		Score:       apiResp.Score,
		Deliverable: apiResp.Deliverable,
		Disposable:  apiResp.Disposable,
		RoleAccount: apiResp.RoleAccount,
		FreeEmail:   apiResp.FreeEmail,
		CatchAll:    apiResp.CatchAll,
		Reason:      apiResp.Reason,
	}, nil
}

// ValidateBatch validates multiple emails (if supported by API)
func (v *MoribouncerValidator) ValidateBatch(ctx context.Context, emails []string) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	for _, email := range emails {
		result, err := v.Validate(ctx, email)
		if err != nil {
			// Continue with other emails even if one fails
			results[email] = &ValidationResult{
				Email:  email,
				Status: "unknown",
				Reason: err.Error(),
			}
			continue
		}
		results[email] = result
	}

	return results, nil
}
