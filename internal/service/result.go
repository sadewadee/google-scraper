package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// ResultService handles result business logic
type ResultService struct {
	results domain.ResultRepository
}

// NewResultService creates a new ResultService
func NewResultService(results domain.ResultRepository) *ResultService {
	return &ResultService{
		results: results,
	}
}

// Create creates a new result
func (s *ResultService) Create(ctx context.Context, jobID uuid.UUID, data []byte) error {
	return s.results.Create(ctx, jobID, data)
}

// CreateBatch creates multiple results
func (s *ResultService) CreateBatch(ctx context.Context, jobID uuid.UUID, data [][]byte) error {
	return s.results.CreateBatch(ctx, jobID, data)
}

// ListAll retrieves all results with pagination (global view)
func (s *ResultService) ListAll(ctx context.Context, limit, offset int) ([][]byte, int, error) {
	return s.results.ListAll(ctx, limit, offset)
}

// ListByJobID retrieves results for a job
func (s *ResultService) ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error) {
	return s.results.ListByJobID(ctx, jobID, limit, offset)
}

// CountByJobID counts results for a job
func (s *ResultService) CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error) {
	return s.results.CountByJobID(ctx, jobID)
}

// StreamByJobID streams results for a job
func (s *ResultService) StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error {
	return s.results.StreamByJobID(ctx, jobID, fn)
}

// GetPlaceStats retrieves place statistics
func (s *ResultService) GetPlaceStats(ctx context.Context) (*domain.PlaceStats, error) {
	return s.results.GetPlaceStats(ctx)
}
