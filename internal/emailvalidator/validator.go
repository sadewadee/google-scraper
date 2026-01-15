package emailvalidator

import "context"

// Validator interface for email validation
type Validator interface {
	Validate(ctx context.Context, email string) (*ValidationResult, error)
}

// ValidationResult from email validation API
type ValidationResult struct {
	Email       string  `json:"email"`
	Status      string  `json:"status"`       // valid, invalid, unknown, catch_all
	Score       float64 `json:"score"`        // 0-100
	Deliverable bool    `json:"deliverable"`
	Disposable  bool    `json:"disposable"`
	RoleAccount bool    `json:"role_account"` // info@, support@, etc.
	FreeEmail   bool    `json:"free_email"`   // gmail, yahoo, etc.
	CatchAll    bool    `json:"catch_all"`    // accepts any email
	Reason      string  `json:"reason"`
}

// ShouldAccept returns true if email is acceptable for leads
func (r *ValidationResult) ShouldAccept() bool {
	return r.Status == "valid" &&
		r.Deliverable &&
		!r.Disposable &&
		!r.RoleAccount &&
		r.Score >= 70
}

// ShouldAcceptRelaxed returns true with more relaxed criteria
// (allows role accounts and catch-all domains)
func (r *ValidationResult) ShouldAcceptRelaxed() bool {
	return (r.Status == "valid" || r.Status == "catch_all") &&
		r.Deliverable &&
		!r.Disposable &&
		r.Score >= 50
}

// NoOpValidator is a validator that accepts all emails (for testing/disabled mode)
type NoOpValidator struct{}

// Validate always returns a valid result for NoOpValidator
func (v *NoOpValidator) Validate(_ context.Context, email string) (*ValidationResult, error) {
	return &ValidationResult{
		Email:       email,
		Status:      "valid",
		Score:       100,
		Deliverable: true,
		Disposable:  false,
	}, nil
}
