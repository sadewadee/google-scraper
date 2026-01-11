package service

import (
	"context"
	"fmt"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// StatsService handles statistics aggregation
type StatsService struct {
	jobs    domain.JobRepository
	workers domain.WorkerRepository
	results domain.ResultRepository
}

// NewStatsService creates a new StatsService
func NewStatsService(
	jobs domain.JobRepository,
	workers domain.WorkerRepository,
	results domain.ResultRepository,
) *StatsService {
	return &StatsService{
		jobs:    jobs,
		workers: workers,
		results: results,
	}
}

// GetStats retrieves aggregated statistics for the dashboard
func (s *StatsService) GetStats(ctx context.Context) (*domain.Stats, error) {
	// Get job stats
	jobStats, err := s.jobs.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}

	// Get worker stats
	workerStats, err := s.workers.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker stats: %w", err)
	}

	// Get place stats
	placeStats, err := s.results.GetPlaceStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get place stats: %w", err)
	}

	return &domain.Stats{
		Jobs:    *jobStats,
		Workers: *workerStats,
		Places:  *placeStats,
	}, nil
}
