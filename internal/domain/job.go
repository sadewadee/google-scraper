package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// CoverageMode determines the geographic search strategy
type CoverageMode string

const (
	// CoverageModeSingle searches from a single point (default, legacy behavior)
	CoverageModeSingle CoverageMode = "single"
	// CoverageModeFull splits the bounding box into a grid of search points based on radius
	CoverageModeFull CoverageMode = "full"
)

// BoundingBox represents a geographic bounding box for area coverage
// Field names match frontend/Nominatim API naming convention
type BoundingBox struct {
	MinLat float64 `json:"min_lat"` // Southern boundary latitude
	MaxLat float64 `json:"max_lat"` // Northern boundary latitude
	MinLon float64 `json:"min_lon"` // Western boundary longitude
	MaxLon float64 `json:"max_lon"` // Eastern boundary longitude
}

// IsValid returns true if the bounding box has valid coordinates
func (b *BoundingBox) IsValid() bool {
	if b == nil {
		return false
	}
	// Check latitude bounds (-90 to 90)
	if b.MaxLat < -90 || b.MaxLat > 90 || b.MinLat < -90 || b.MinLat > 90 {
		return false
	}
	// Check longitude bounds (-180 to 180)
	if b.MaxLon < -180 || b.MaxLon > 180 || b.MinLon < -180 || b.MinLon > 180 {
		return false
	}
	// Max must be greater than Min for latitude
	if b.MaxLat <= b.MinLat {
		return false
	}
	// Max must be greater than Min for longitude (cross-dateline not supported)
	if b.MaxLon <= b.MinLon {
		return false
	}
	return true
}

// Center returns the center point of the bounding box
func (b *BoundingBox) Center() (lat, lon float64) {
	lat = (b.MaxLat + b.MinLat) / 2
	lon = (b.MaxLon + b.MinLon) / 2
	return
}

// GridPoint represents a single point in the search grid
type GridPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// GenerateGrid creates a grid of search points within the bounding box
// gridSize determines the number of points per side (e.g., 3 = 3x3 = 9 points)
func (b *BoundingBox) GenerateGrid(gridSize int) []GridPoint {
	if gridSize < 1 {
		gridSize = 1
	}
	if gridSize > 10 {
		gridSize = 10 // Cap at 10x10 = 100 points max
	}

	points := make([]GridPoint, 0, gridSize*gridSize)

	latStep := (b.MaxLat - b.MinLat) / float64(gridSize)
	lonStep := (b.MaxLon - b.MinLon) / float64(gridSize)

	// Generate grid points (center of each cell)
	for i := 0; i < gridSize; i++ {
		for j := 0; j < gridSize; j++ {
			lat := b.MinLat + latStep*(float64(i)+0.5)
			lon := b.MinLon + lonStep*(float64(j)+0.5)
			points = append(points, GridPoint{Lat: lat, Lon: lon})
		}
	}

	return points
}

// GenerateGridByRadius creates a grid of search points within the bounding box
// based on the radius in meters (distance between grid points)
func (b *BoundingBox) GenerateGridByRadius(radiusMeters int) []GridPoint {
	if radiusMeters < 100 {
		radiusMeters = 100
	}

	// Convert radius to degrees (approximate)
	// 1 degree latitude ≈ 111320 meters
	latStep := float64(radiusMeters) / 111320.0

	// Longitude varies with latitude, use center latitude for approximation
	centerLat := (b.MaxLat + b.MinLat) / 2
	lonStep := float64(radiusMeters) / (111320.0 * math.Cos(centerLat*math.Pi/180.0))

	points := make([]GridPoint, 0)

	// Generate grid points
	for lat := b.MinLat; lat <= b.MaxLat; lat += latStep {
		for lon := b.MinLon; lon <= b.MaxLon; lon += lonStep {
			points = append(points, GridPoint{Lat: lat, Lon: lon})
		}
	}

	// Cap at 100 points max
	if len(points) > 100 {
		points = points[:100]
	}

	return points
}

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

	// Geo coverage settings for area-wide scraping
	LocationName string       `json:"location_name,omitempty"` // Human-readable location name
	BoundingBox  *BoundingBox `json:"boundingbox,omitempty"`
	CoverageMode CoverageMode `json:"coverage_mode,omitempty"`
	GridPoints   int          `json:"grid_points,omitempty"` // Number of grid points generated
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

	// Geo coverage settings for area-wide scraping
	LocationName string       `json:"location_name,omitempty"`
	BoundingBox  *BoundingBox `json:"boundingbox,omitempty"`
	CoverageMode CoverageMode `json:"coverage_mode,omitempty"`
}

// EstimateTotalPlaces estimates total places based on job config
// For full coverage mode, multiplies by number of grid points
func (r *CreateJobRequest) EstimateTotalPlaces() int {
	if len(r.Keywords) == 0 {
		return 0
	}
	depth := r.Depth
	if depth == 0 {
		depth = 10 // default depth
	}

	// Base estimate: each scroll shows ~20 results, depth is number of scrolls
	resultsPerKeyword := depth * 20

	// For full coverage mode, multiply by number of grid points
	gridMultiplier := 1
	if r.CoverageMode == CoverageModeFull && r.BoundingBox != nil && r.BoundingBox.IsValid() {
		radius := r.Radius
		if radius == 0 {
			radius = 5000 // default 5km
		}
		gridPoints := r.BoundingBox.GenerateGridByRadius(radius)
		gridMultiplier = len(gridPoints)
		if gridMultiplier < 1 {
			gridMultiplier = 1
		}
	}

	return len(r.Keywords) * resultsPerKeyword * gridMultiplier
}

// CalculateGridPoints returns the number of grid points for this request
func (r *CreateJobRequest) CalculateGridPoints() int {
	if r.CoverageMode != CoverageModeFull || r.BoundingBox == nil || !r.BoundingBox.IsValid() {
		return 1
	}
	radius := r.Radius
	if radius == 0 {
		radius = 5000
	}
	return len(r.BoundingBox.GenerateGridByRadius(radius))
}

// ToJob converts a CreateJobRequest to a Job
func (r *CreateJobRequest) ToJob() *Job {
	now := time.Now().UTC()

	// Set default coverage mode
	coverageMode := r.CoverageMode
	if coverageMode == "" {
		coverageMode = CoverageModeSingle
	}

	// Calculate grid points for full coverage mode
	gridPoints := 1
	if coverageMode == CoverageModeFull && r.BoundingBox != nil && r.BoundingBox.IsValid() {
		gridPoints = r.CalculateGridPoints()
	}

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
		LocationName: r.LocationName,
		BoundingBox:  r.BoundingBox,
		CoverageMode: coverageMode,
		GridPoints:   gridPoints,
	}

	// Set defaults
	if config.Lang == "" {
		config.Lang = "en"
	}
	if config.Zoom == 0 {
		config.Zoom = 15
	}
	if config.Radius == 0 {
		config.Radius = 5000
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
