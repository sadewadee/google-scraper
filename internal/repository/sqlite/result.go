package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sadewadee/google-scraper/internal/domain"
)

// ResultRepository implements domain.ResultRepository for SQLite
type ResultRepository struct {
	db *sql.DB
}

// NewResultRepository creates a new ResultRepository
func NewResultRepository(db *sql.DB) *ResultRepository {
	return &ResultRepository{db: db}
}

// Create creates a new result
func (r *ResultRepository) Create(ctx context.Context, jobID uuid.UUID, data []byte) error {
	query := `INSERT INTO results (job_id, data, created_at) VALUES (?, ?, ?)`
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query, jobID.String(), string(data), now)
	return err
}

// CreateBatch creates multiple results in a batch
func (r *ResultRepository) CreateBatch(ctx context.Context, jobID uuid.UUID, data [][]byte) error {
	if len(data) == 0 {
		return nil
	}

	// SQLite has limit on number of variables. Split into chunks if necessary.
	// Safe batch size: 100
	batchSize := 100
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*3)
		now := time.Now().UTC().Format(time.RFC3339)
		jobIDStr := jobID.String()

		for _, d := range batch {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			valueArgs = append(valueArgs, jobIDStr, string(d), now)
		}

		query := fmt.Sprintf("INSERT INTO results (job_id, data, created_at) VALUES %s",
			strings.Join(valueStrings, ","))

		_, err := r.db.ExecContext(ctx, query, valueArgs...)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListByJobID retrieves results for a job with pagination
func (r *ResultRepository) ListByJobID(ctx context.Context, jobID uuid.UUID, limit, offset int) ([][]byte, int, error) {
	// First get total count
	countQuery := `SELECT COUNT(*) FROM results WHERE job_id = ?`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, jobID.String()).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get results
	query := `SELECT data FROM results WHERE job_id = ? ORDER BY id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, jobID.String(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results [][]byte
	for rows.Next() {
		var dataStr string
		if err := rows.Scan(&dataStr); err != nil {
			return nil, 0, err
		}
		results = append(results, []byte(dataStr))
	}

	return results, total, rows.Err()
}

// CountByJobID counts results for a job
func (r *ResultRepository) CountByJobID(ctx context.Context, jobID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM results WHERE job_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, jobID.String()).Scan(&count)
	return count, err
}

// DeleteByJobID deletes all results for a job
func (r *ResultRepository) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	query := `DELETE FROM results WHERE job_id = ?`
	_, err := r.db.ExecContext(ctx, query, jobID.String())
	return err
}

// GetPlaceStats retrieves place statistics
func (r *ResultRepository) GetPlaceStats(ctx context.Context) (*domain.PlaceStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN date(created_at) = date('now') THEN 1 ELSE 0 END) as today
		FROM results
	`

	stats := &domain.PlaceStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(&stats.TotalScraped, &stats.Today)
	return stats, err
}

// StreamByJobID streams results for a job
func (r *ResultRepository) StreamByJobID(ctx context.Context, jobID uuid.UUID, fn func(data []byte) error) error {
	query := `SELECT data FROM results WHERE job_id = ? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, query, jobID.String())
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var dataStr string
		if err := rows.Scan(&dataStr); err != nil {
			return err
		}
		if err := fn([]byte(dataStr)); err != nil {
			return err
		}
	}

	return rows.Err()
}
