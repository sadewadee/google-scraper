package runner

import (
	"fmt"
	"strings"

	"github.com/gosom/scrapemate"
	"github.com/sadewadee/google-scraper/deduper"
	"github.com/sadewadee/google-scraper/exiter"
	"github.com/sadewadee/google-scraper/internal/emailvalidator"
)

// SeedJobConfig for creating seed jobs from API
type SeedJobConfig struct {
	Keywords       []string
	FastMode       bool
	LangCode       string
	Depth          int
	Email          bool
	GeoCoordinates string // "lat,lon" or ""
	Zoom           int
	Radius         float64
	ExtraReviews   bool
	Dedup          deduper.Deduper
	ExitMonitor    exiter.Exiter
	EmailValidator emailvalidator.Validator
}

// CreateSeedJobsFromKeywords creates seed jobs from a slice of keywords.
// This is a reusable wrapper for CreateSeedJobs that accepts []string instead of io.Reader.
// Used by both CLI and API (Dashboard).
func CreateSeedJobsFromKeywords(cfg SeedJobConfig) ([]scrapemate.IJob, error) {
	if len(cfg.Keywords) == 0 {
		return nil, fmt.Errorf("at least one keyword is required")
	}

	// Convert []string to io.Reader (adapter pattern)
	input := strings.NewReader(strings.Join(cfg.Keywords, "\n"))

	return CreateSeedJobs(
		cfg.FastMode,
		cfg.LangCode,
		input,
		cfg.Depth,
		cfg.Email,
		cfg.GeoCoordinates,
		cfg.Zoom,
		cfg.Radius,
		cfg.Dedup,
		cfg.ExitMonitor,
		cfg.EmailValidator,
		cfg.ExtraReviews,
	)
}

// FormatGeoCoordinates formats latitude and longitude into a string.
// Returns empty string if both are zero.
func FormatGeoCoordinates(lat, lon float64) string {
	if lat == 0 && lon == 0 {
		return ""
	}
	return fmt.Sprintf("%f,%f", lat, lon)
}
