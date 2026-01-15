package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/domain"
)

const (
	// Query timeouts for result repository
	resultQueryTimeout     = 30 * time.Second
	resultCountTimeout     = 10 * time.Second
	resultStreamTimeout    = 5 * time.Minute // Longer for streaming large datasets
	resultStatsTimeout     = 15 * time.Second
)

// ResultRepository implements domain.ResultRepository for PostgreSQL
type ResultRepository struct {
	db *sql.DB
}

// NewResultRepository creates a new ResultRepository
func NewResultRepository(db *sql.DB) *ResultRepository {
	return &ResultRepository{db: db}
}

// Create creates a new result
func (r *ResultRepository) Create(ctx context.Context, jobID uuid.UUID, data []byte) error {
	query := `INSERT INTO results (job_id, data) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, jobID, data)
	return err
}

// CreateBatch creates multiple results in a batch
func (r *ResultRepository) CreateBatch(ctx context.Context, jobID uuid.UUID, data [][]byte) error {
	if len(data) == 0 {
		return nil
	}

	// Build batch insert query
	values := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, jobID)

	for i, d := range data {
		values = append(values, fmt.Sprintf("($1, $%d)", i+2))
		args = append(args, d)
	}

	query := fmt.Sprintf(`
		INSERT INTO results (job_id, data) VALUES %s
		ON CONFLICT DO NOTHING
	`, strings.Join(values, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// ListAll retrieves all results with pagination (global view)
func (r *ResultRepository) ListAll(ctx context.Context, limit, offset int) ([][]byte, int, error) {
	// Use approximate count for global queries (much faster on large tables)
	// pg_class.reltuples is updated by VACUUM/ANALYZE and is usually accurate enough
	countCtx, countCancel := context.WithTimeout(ctx, resultCountTimeout)
	defer countCancel()

	countQuery := `
		SELECT COALESCE(
			(SELECT reltuples::bigint FROM pg_class WHERE relname = 'results'),
			(SELECT COUNT(*) FROM results)
		)::int
	`
	var total int
	err := r.db.QueryRowContext(countCtx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Get results with timeout
	queryCtx, queryCancel := context.WithTimeout(ctx, resultQueryTimeout)
	defer queryCancel()

	query := `
		SELECT data FROM results
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(queryCtx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var results [][]byte
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		results = append(results, data)
	}

	return results, total, rows.Err()
}

// ListByJobID retrieves results for a job with pagination
func (r *ResultRepository) ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error) {
	// Get total count with timeout
	countCtx, countCancel := context.WithTimeout(ctx, resultCountTimeout)
	defer countCancel()

	countQuery := `SELECT COUNT(*) FROM results WHERE job_id = $1`
	var total int
	err := r.db.QueryRowContext(countCtx, countQuery, jobID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Get results with timeout
	queryCtx, queryCancel := context.WithTimeout(ctx, resultQueryTimeout)
	defer queryCancel()

	query := `
		SELECT data FROM results
		WHERE job_id = $1
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(queryCtx, query, jobID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var results [][]byte
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		results = append(results, data)
	}

	return results, total, rows.Err()
}

// CountByJobID counts results for a job
func (r *ResultRepository) CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error) {
	countCtx, cancel := context.WithTimeout(ctx, resultCountTimeout)
	defer cancel()

	query := `SELECT COUNT(*) FROM results WHERE job_id = $1`
	var count int
	err := r.db.QueryRowContext(countCtx, query, jobID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}
	return count, nil
}

// DeleteByJobID deletes all results for a job
func (r *ResultRepository) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	query := `DELETE FROM results WHERE job_id = $1`
	_, err := r.db.ExecContext(ctx, query, jobID)
	return err
}

// GetPlaceStats retrieves place statistics
func (r *ResultRepository) GetPlaceStats(ctx context.Context) (*domain.PlaceStats, error) {
	statsCtx, cancel := context.WithTimeout(ctx, resultStatsTimeout)
	defer cancel()

	// Use approximate count for total (faster), exact count for today
	query := `
		SELECT
			COALESCE(
				(SELECT reltuples::bigint FROM pg_class WHERE relname = 'results'),
				(SELECT COUNT(*) FROM results WHERE job_id IS NOT NULL)
			)::int as total,
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as today
		FROM results
		WHERE job_id IS NOT NULL AND created_at >= CURRENT_DATE
	`

	stats := &domain.PlaceStats{}
	err := r.db.QueryRowContext(statsCtx, query).Scan(&stats.TotalScraped, &stats.Today)
	if err != nil {
		return nil, fmt.Errorf("stats query failed: %w", err)
	}
	return stats, nil
}

// StreamByJobID streams results for export (memory efficient)
// Uses a longer timeout (5 min) to handle large datasets
func (r *ResultRepository) StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error {
	streamCtx, cancel := context.WithTimeout(ctx, resultStreamTimeout)
	defer cancel()

	query := `SELECT data FROM results WHERE job_id = $1 ORDER BY id ASC`

	rows, err := r.db.QueryContext(streamCtx, query, jobID)
	if err != nil {
		return fmt.Errorf("stream query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		// Check context between rows to allow early cancellation
		select {
		case <-streamCtx.Done():
			return streamCtx.Err()
		default:
		}

		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		if err := fn(data); err != nil {
			return err
		}
	}

	return rows.Err()
}
