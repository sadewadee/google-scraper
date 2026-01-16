package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// IsTerminal returns true if the job is in a terminal state
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusCancelled
}

// CanPause returns true if the job can be paused
func (s JobStatus) CanPause() bool {
	return s == JobStatusRunning || s == JobStatusQueued
}

// CanResume returns true if the job can be resumed
func (s JobStatus) CanResume() bool {
	return s == JobStatusPaused
}

// CanCancel returns true if the job can be cancelled
func (s JobStatus) CanCancel() bool {
	return s == JobStatusPending || s == JobStatusQueued || s == JobStatusRunning || s == JobStatusPaused
}

// Job represents a scraping job in the queue
type Job struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Status   JobStatus `json:"status"`
	Priority int       `json:"priority"`

	// Configuration
	Config JobConfig `json:"config"`

	// Progress
	Progress JobProgress `json:"progress"`

	// Worker assignment
	WorkerID *string `json:"worker_id,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error info
	ErrorMessage *string `json:"error_message,omitempty"`
}

// JobConfig contains the scraping configuration
type JobConfig struct {
	Keywords     []string      `json:"keywords"`
	Lang         string        `json:"lang"`
	GeoLat       *float64      `json:"geo_lat,omitempty"`
	GeoLon       *float64      `json:"geo_lon,omitempty"`
	Zoom         int           `json:"zoom"`
	Radius       int           `json:"radius"`
	Depth        int           `json:"depth"`
	FastMode     bool          `json:"fast_mode"`
	ExtractEmail bool          `json:"extract_email"`
	MaxTime      time.Duration `json:"max_time"`
	Proxies      []string      `json:"proxies,omitempty"`
}

// JobProgress tracks the scraping progress
type JobProgress struct {
	TotalPlaces   int     `json:"total_places"`
	ScrapedPlaces int     `json:"scraped_places"`
	FailedPlaces  int     `json:"failed_places"`
	Percentage    float64 `json:"percentage"`
}

// CalculatePercentage updates the percentage based on scraped/total
func (p *JobProgress) CalculatePercentage() {
	if p.TotalPlaces > 0 {
		p.Percentage = float64(p.ScrapedPlaces) / float64(p.TotalPlaces) * 100
	} else {
		p.Percentage = 0
	}
}

// CreateJobRequest is the request to create a new job
type CreateJobRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=255"`
	Keywords     []string `json:"keywords" validate:"required,min=1,dive,min=1"`
	Lang         string   `json:"lang" validate:"required,len=2"`
	GeoLat       *float64 `json:"geo_lat,omitempty" validate:"omitempty,latitude"`
	GeoLon       *float64 `json:"geo_lon,omitempty" validate:"omitempty,longitude"`
	Zoom         int      `json:"zoom" validate:"min=1,max=21"`
	Radius       int      `json:"radius" validate:"min=0"`
	Depth        int      `json:"depth" validate:"required,min=1,max=100"`
	FastMode     bool     `json:"fast_mode"`
	ExtractEmail bool     `json:"extract_email"`
	MaxTime      int      `json:"max_time" validate:"required,min=180"` // seconds
	Proxies      []string `json:"proxies,omitempty"`
	Priority     int      `json:"priority" validate:"min=0,max=100"`
}

// EstimateTotalPlaces estimates total places based on job config
// Each keyword typically yields approximately depth results
func (r *CreateJobRequest) EstimateTotalPlaces() int {
	if len(r.Keywords) == 0 {
		return 0
	}
	depth := r.Depth
	if depth == 0 {
		depth = 10 // default depth
	}
	return len(r.Keywords) * depth
}

// ToJob converts a CreateJobRequest to a Job
func (r *CreateJobRequest) ToJob() *Job {
	now := time.Now().UTC()

	config := JobConfig{
		Keywords:     r.Keywords,
		Lang:         r.Lang,
		GeoLat:       r.GeoLat,
		GeoLon:       r.GeoLon,
		Zoom:         r.Zoom,
		Radius:       r.Radius,
		Depth:        r.Depth,
		FastMode:     r.FastMode,
		ExtractEmail: r.ExtractEmail,
		MaxTime:      time.Duration(r.MaxTime) * time.Second,
		Proxies:      r.Proxies,
	}

	// Set defaults
	if config.Lang == "" {
		config.Lang = "en"
	}
	if config.Zoom == 0 {
		config.Zoom = 15
	}
	if config.Radius == 0 {
		config.Radius = 10000
	}
	if config.Depth == 0 {
		config.Depth = 10
	}
	if config.MaxTime == 0 {
		config.MaxTime = 10 * time.Minute
	}

	return &Job{
		ID:       uuid.New(),
		Name:     r.Name,
		Status:   JobStatusPending,
		Priority: r.Priority,
		Config:   config,
		Progress: JobProgress{
			TotalPlaces:   r.EstimateTotalPlaces(),
			ScrapedPlaces: 0,
			FailedPlaces:  0,
			Percentage:    0,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateJobRequest is the request to update a job (pause/resume/cancel)
type UpdateJobRequest struct {
	Status *JobStatus `json:"status,omitempty"`
}

// JobListParams are parameters for listing jobs
type JobListParams struct {
	Status   *JobStatus
	WorkerID *string
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string
}
