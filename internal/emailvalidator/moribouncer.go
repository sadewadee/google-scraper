package emailvalidator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProxyProvider is a function that returns a proxy for SMTP checks
// Returns host, port, or empty/0 if no proxy available
type ProxyProvider func(ctx context.Context) (host string, port int, err error)

// MordibouncerValidator validates emails using Mordibouncer API
// (Note: The struct name intentionally matches the API service name "Mordibouncer")
type MordibouncerValidator struct {
	apiURL        string
	apiKey        string
	httpClient    *http.Client
	proxyProvider ProxyProvider
}

// ProxyConfig for SOCKS5 proxy support in Mordibouncer requests
type ProxyConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// MordibouncerConfig for Mordibouncer validator
type MordibouncerConfig struct {
	APIURL        string        // e.g., "https://mailexchange.kremlit.dev"
	APIKey        string        // x-mordibouncer-secret header value
	Timeout       time.Duration
	ProxyProvider ProxyProvider // Optional: function to get proxy from database
}

// freeEmailDomains contains common free email providers
var freeEmailDomains = map[string]bool{
	"gmail.com": true, "yahoo.com": true, "hotmail.com": true,
	"outlook.com": true, "aol.com": true, "icloud.com": true,
	"protonmail.com": true, "mail.com": true, "zoho.com": true,
	"yandex.com": true, "gmx.com": true, "live.com": true,
}

// NewMordibouncerValidator creates a new Mordibouncer validator
func NewMordibouncerValidator(cfg MordibouncerConfig) *MordibouncerValidator {
	timeout := cfg.Timeout
	if timeout == 0 {
		// SMTP checks can take time due to:
		// - Multiple MX record attempts
		// - Greylisting delays
		// - Connection timeouts to remote mail servers
		timeout = 60 * time.Second
	}

	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://mailexchange.kremlit.dev"
	}
	// Remove trailing slash if present
	apiURL = strings.TrimSuffix(apiURL, "/")

	return &MordibouncerValidator{
		apiURL: apiURL,
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		proxyProvider: cfg.ProxyProvider,
	}
}

// Config is an alias for backward compatibility
// Deprecated: Use MordibouncerConfig instead
type Config = MordibouncerConfig

// NewMoribouncerValidator is an alias for backward compatibility
// Deprecated: Use NewMordibouncerValidator instead
func NewMoribouncerValidator(cfg Config) *MordibouncerValidator {
	return NewMordibouncerValidator(cfg)
}

// mordibouncerRequest is the API request format for Mordibouncer
type mordibouncerRequest struct {
	ToEmail string       `json:"to_email"`
	Proxy   *ProxyConfig `json:"proxy,omitempty"`
}

// mordibouncerResponse is the API response format from Mordibouncer
type mordibouncerResponse struct {
	Input       string `json:"input"`
	IsReachable string `json:"is_reachable"` // safe, risky, invalid, unknown
	Misc        struct {
		IsDisposable   bool    `json:"is_disposable"`
		IsRoleAccount  bool    `json:"is_role_account"`
		IsB2C          bool    `json:"is_b2c"`
		GravatarURL    *string `json:"gravatar_url"`
		HaveIBeenPwned *bool   `json:"haveibeenpwned"`
	} `json:"misc"`
	MX struct {
		AcceptsMail bool     `json:"accepts_mail"`
		Records     []string `json:"records"`
	} `json:"mx"`
	SMTP struct {
		CanConnectSMTP bool `json:"can_connect_smtp"`
		HasFullInbox   bool `json:"has_full_inbox"`
		IsCatchAll     bool `json:"is_catch_all"`
		IsDeliverable  bool `json:"is_deliverable"`
		IsDisabled     bool `json:"is_disabled"`
	} `json:"smtp"`
	Syntax struct {
		Address         string  `json:"address"`
		Domain          string  `json:"domain"`
		IsValidSyntax   bool    `json:"is_valid_syntax"`
		Username        string  `json:"username"`
		NormalizedEmail string  `json:"normalized_email"`
		Suggestion      *string `json:"suggestion"`
	} `json:"syntax"`
	Debug struct {
		BackendName string `json:"backend_name"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
	} `json:"debug"`
}

// Validate validates a single email using Mordibouncer API
func (v *MordibouncerValidator) Validate(ctx context.Context, email string) (*ValidationResult, error) {
	// Build request body
	reqBody := mordibouncerRequest{
		ToEmail: email,
	}

	// Get proxy from provider if available
	if v.proxyProvider != nil {
		host, port, err := v.proxyProvider(ctx)
		if err == nil && host != "" && port > 0 {
			reqBody.Proxy = &ProxyConfig{
				Host: host,
				Port: port,
			}
		}
		// If proxy provider fails, continue without proxy
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request URL: POST /v0/check_email
	reqURL := fmt.Sprintf("%s/v0/check_email", v.apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-mordibouncer-secret", v.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Mordibouncer API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error body for better debugging
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if len(errBody) > 0 {
			return nil, fmt.Errorf("Mordibouncer API returned status %d: %s", resp.StatusCode, string(errBody))
		}
		return nil, fmt.Errorf("Mordibouncer API returned status %d", resp.StatusCode)
	}

	var apiResp mordibouncerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Map Mordibouncer response to ValidationResult
	return v.mapResponse(email, &apiResp), nil
}

// mapResponse converts Mordibouncer API response to ValidationResult
// Score calculation logic:
// - Base score from is_reachable: safe=100, risky=60, invalid=0, unknown=30
// - Penalties: disposable(-80), role_account(-50), catch_all(-30), full_inbox(-60), disabled=0
func (v *MordibouncerValidator) mapResponse(inputEmail string, resp *mordibouncerResponse) *ValidationResult {
	// Map is_reachable to status and score
	var status string
	var score float64

	switch resp.IsReachable {
	case "safe":
		status = "valid"
		score = 100
	case "risky":
		status = "risky"
		score = 60
	case "invalid":
		status = "invalid"
		score = 0
	case "unknown":
		status = "unknown"
		score = 30
	default:
		status = "unknown"
		score = 30
	}

	// Adjust score based on additional factors
	if resp.Misc.IsDisposable {
		score = min(score, 20)
	}
	if resp.Misc.IsRoleAccount {
		score = min(score, 50)
	}
	if resp.SMTP.IsCatchAll {
		score = min(score, 70)
	}
	if resp.SMTP.HasFullInbox {
		score = min(score, 40)
	}
	if resp.SMTP.IsDisabled {
		score = 0
		status = "invalid"
	}

	// Determine reason
	var reason string
	if !resp.Syntax.IsValidSyntax {
		reason = "invalid syntax"
	} else if !resp.MX.AcceptsMail {
		reason = "domain does not accept mail"
	} else if !resp.SMTP.CanConnectSMTP {
		reason = "cannot connect to SMTP server"
	} else if !resp.SMTP.IsDeliverable {
		reason = "not deliverable"
	} else if resp.SMTP.IsDisabled {
		reason = "mailbox disabled"
	} else if resp.SMTP.HasFullInbox {
		reason = "mailbox full"
	} else if resp.Misc.IsDisposable {
		reason = "disposable email"
	} else if resp.Misc.IsRoleAccount {
		reason = "role account"
	} else if resp.SMTP.IsCatchAll {
		reason = "catch-all domain"
	}

	// Use normalized email if available, otherwise fall back to input
	email := resp.Syntax.NormalizedEmail
	if email == "" {
		email = resp.Input
	}
	if email == "" {
		email = inputEmail
	}

	isFreeEmail := freeEmailDomains[resp.Syntax.Domain]

	return &ValidationResult{
		Email:       email,
		Status:      status,
		Score:       score,
		Deliverable: resp.SMTP.IsDeliverable && resp.IsReachable == "safe",
		Disposable:  resp.Misc.IsDisposable,
		RoleAccount: resp.Misc.IsRoleAccount,
		FreeEmail:   isFreeEmail,
		CatchAll:    resp.SMTP.IsCatchAll,
		Reason:      reason,
	}
}

// ValidateBatch validates multiple emails
func (v *MordibouncerValidator) ValidateBatch(ctx context.Context, emails []string) (map[string]*ValidationResult, error) {
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
