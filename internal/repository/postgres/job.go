package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/gosom/google-maps-scraper/internal/domain"
)

// JobRepository implements domain.JobRepository for PostgreSQL
type JobRepository struct {
	db *sql.DB
}

// NewJobRepository creates a new JobRepository
func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create creates a new job
func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	query := `
		INSERT INTO jobs_queue (
			id, name, status, priority,
			keywords, lang, geo_lat, geo_lon, zoom, radius, depth,
			fast_mode, extract_email, max_time, proxies,
			total_places, scraped_places, failed_places,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Status, job.Priority,
		pq.Array(job.Config.Keywords), job.Config.Lang, job.Config.GeoLat, job.Config.GeoLon,
		job.Config.Zoom, job.Config.Radius, job.Config.Depth,
		job.Config.FastMode, job.Config.ExtractEmail, job.Config.MaxTime, pq.Array(job.Config.Proxies),
		job.Progress.TotalPlaces, job.Progress.ScrapedPlaces, job.Progress.FailedPlaces,
		job.CreatedAt, job.UpdatedAt,
	)

	return err
}

// GetByID retrieves a job by ID
func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	query := `
		SELECT
			id, name, status, priority,
			keywords, lang, geo_lat, geo_lon, zoom, radius, depth,
			fast_mode, extract_email, max_time, proxies,
			total_places, scraped_places, failed_places,
			worker_id, created_at, updated_at, started_at, completed_at,
			error_message
		FROM jobs_queue
		WHERE id = $1
	`

	job := &domain.Job{}
	var keywords, proxies pq.StringArray
	var maxTime time.Duration

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Name, &job.Status, &job.Priority,
		&keywords, &job.Config.Lang, &job.Config.GeoLat, &job.Config.GeoLon,
		&job.Config.Zoom, &job.Config.Radius, &job.Config.Depth,
		&job.Config.FastMode, &job.Config.ExtractEmail, &maxTime, &proxies,
		&job.Progress.TotalPlaces, &job.Progress.ScrapedPlaces, &job.Progress.FailedPlaces,
		&job.WorkerID, &job.CreatedAt, &job.UpdatedAt, &job.StartedAt, &job.CompletedAt,
		&job.ErrorMessage,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	job.Config.Keywords = keywords
	job.Config.Proxies = proxies
	job.Config.MaxTime = maxTime
	job.Progress.CalculatePercentage()

	return job, nil
}

// List retrieves jobs with optional filtering
func (r *JobRepository) List(ctx context.Context, params domain.JobListParams) ([]*domain.Job, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	if params.WorkerID != nil {
		conditions = append(conditions, fmt.Sprintf("worker_id = $%d", argIdx))
		args = append(args, *params.WorkerID)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs_queue %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Order
	orderBy := "created_at"
	if params.OrderBy != "" {
		orderBy = params.OrderBy
	}
	orderDir := "DESC"
	if params.OrderDir != "" {
		orderDir = params.OrderDir
	}

	// Limit & offset
	limit := 20
	if params.Limit > 0 {
		limit = params.Limit
	}
	offset := params.Offset

	// Main query
	query := fmt.Sprintf(`
		SELECT
			id, name, status, priority,
			keywords, lang, geo_lat, geo_lon, zoom, radius, depth,
			fast_mode, extract_email, max_time, proxies,
			total_places, scraped_places, failed_places,
			worker_id, created_at, updated_at, started_at, completed_at,
			error_message
		FROM jobs_queue
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, orderDir, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job := &domain.Job{}
		var keywords, proxies pq.StringArray
		var maxTime time.Duration

		err := rows.Scan(
			&job.ID, &job.Name, &job.Status, &job.Priority,
			&keywords, &job.Config.Lang, &job.Config.GeoLat, &job.Config.GeoLon,
			&job.Config.Zoom, &job.Config.Radius, &job.Config.Depth,
			&job.Config.FastMode, &job.Config.ExtractEmail, &maxTime, &proxies,
			&job.Progress.TotalPlaces, &job.Progress.ScrapedPlaces, &job.Progress.FailedPlaces,
			&job.WorkerID, &job.CreatedAt, &job.UpdatedAt, &job.StartedAt, &job.CompletedAt,
			&job.ErrorMessage,
		)
		if err != nil {
			return nil, 0, err
		}

		job.Config.Keywords = keywords
		job.Config.Proxies = proxies
		job.Config.MaxTime = maxTime
		job.Progress.CalculatePercentage()

		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	query := `
		UPDATE jobs_queue SET
			name = $2, status = $3, priority = $4,
			keywords = $5, lang = $6, geo_lat = $7, geo_lon = $8,
			zoom = $9, radius = $10, depth = $11,
			fast_mode = $12, extract_email = $13, max_time = $14, proxies = $15,
			total_places = $16, scraped_places = $17, failed_places = $18,
			worker_id = $19, started_at = $20, completed_at = $21,
			error_message = $22
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Status, job.Priority,
		pq.Array(job.Config.Keywords), job.Config.Lang, job.Config.GeoLat, job.Config.GeoLon,
		job.Config.Zoom, job.Config.Radius, job.Config.Depth,
		job.Config.FastMode, job.Config.ExtractEmail, job.Config.MaxTime, pq.Array(job.Config.Proxies),
		job.Progress.TotalPlaces, job.Progress.ScrapedPlaces, job.Progress.FailedPlaces,
		job.WorkerID, job.StartedAt, job.CompletedAt,
		job.ErrorMessage,
	)

	return err
}

// Delete deletes a job by ID
func (r *JobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM jobs_queue WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateStatus updates only the status of a job
func (r *JobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error {
	var query string

	switch status {
	case domain.JobStatusRunning:
		query = `UPDATE jobs_queue SET status = $2, started_at = NOW() WHERE id = $1`
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled:
		query = `UPDATE jobs_queue SET status = $2, completed_at = NOW(), worker_id = NULL WHERE id = $1`
	default:
		query = `UPDATE jobs_queue SET status = $2 WHERE id = $1`
	}

	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

// UpdateProgress updates the progress of a job
func (r *JobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress domain.JobProgress) error {
	query := `
		UPDATE jobs_queue SET
			total_places = $2,
			scraped_places = $3,
			failed_places = $4
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id,
		progress.TotalPlaces, progress.ScrapedPlaces, progress.FailedPlaces)
	return err
}

// ClaimJob claims a pending job for a worker (atomic operation)
func (r *JobRepository) ClaimJob(ctx context.Context, workerID string) (*domain.Job, error) {
	query := `
		UPDATE jobs_queue SET
			status = 'running',
			worker_id = $1,
			started_at = NOW()
		WHERE id = (
			SELECT id FROM jobs_queue
			WHERE status = 'pending'
			ORDER BY priority DESC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id
	`

	var jobID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, workerID).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // No pending jobs
	}
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, jobID)
}

// ReleaseJob releases a job back to pending status
func (r *JobRepository) ReleaseJob(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE jobs_queue SET
			status = 'pending',
			worker_id = NULL,
			started_at = NULL
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetStats retrieves job statistics
func (r *JobRepository) GetStats(ctx context.Context) (*domain.JobStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'queued') as queued,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'paused') as paused,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
		FROM jobs_queue
	`

	stats := &domain.JobStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.Total, &stats.Pending, &stats.Queued, &stats.Running,
		&stats.Paused, &stats.Completed, &stats.Failed, &stats.Cancelled,
	)

	return stats, err
}
