package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/google-maps-scraper/internal/domain"
)

// JobRepository implements domain.JobRepository for SQLite
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
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?
		)
	`

	keywordsJSON, err := json.Marshal(job.Config.Keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}

	proxiesJSON, err := json.Marshal(job.Config.Proxies)
	if err != nil {
		return fmt.Errorf("failed to marshal proxies: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		job.ID.String(), job.Name, job.Status, job.Priority,
		string(keywordsJSON), job.Config.Lang, job.Config.GeoLat, job.Config.GeoLon,
		job.Config.Zoom, job.Config.Radius, job.Config.Depth,
		job.Config.FastMode, job.Config.ExtractEmail, job.Config.MaxTime.String(), string(proxiesJSON),
		job.Progress.TotalPlaces, job.Progress.ScrapedPlaces, job.Progress.FailedPlaces,
		job.CreatedAt.Format(time.RFC3339), job.UpdatedAt.Format(time.RFC3339),
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
		WHERE id = ?
	`

	job := &domain.Job{}
	var idStr, statusStr string
	var keywordsJSON, proxiesJSON string
	var maxTimeStr string
	var workerID sql.NullString
	var createdAtStr, updatedAtStr string
	var startedAtStr, completedAtStr sql.NullString
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&idStr, &job.Name, &statusStr, &job.Priority,
		&keywordsJSON, &job.Config.Lang, &job.Config.GeoLat, &job.Config.GeoLon,
		&job.Config.Zoom, &job.Config.Radius, &job.Config.Depth,
		&job.Config.FastMode, &job.Config.ExtractEmail, &maxTimeStr, &proxiesJSON,
		&job.Progress.TotalPlaces, &job.Progress.ScrapedPlaces, &job.Progress.FailedPlaces,
		&workerID, &createdAtStr, &updatedAtStr, &startedAtStr, &completedAtStr,
		&errorMessage,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse UUID
	job.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job ID: %w", err)
	}

	job.Status = domain.JobStatus(statusStr)

	// Parse JSON fields
	if err := json.Unmarshal([]byte(keywordsJSON), &job.Config.Keywords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
	}

	if proxiesJSON != "" {
		if err := json.Unmarshal([]byte(proxiesJSON), &job.Config.Proxies); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proxies: %w", err)
		}
	}

	// Parse Duration
	job.Config.MaxTime, err = time.ParseDuration(maxTimeStr)
	if err != nil {
		// Fallback for backward compatibility or if stored as int
		job.Config.MaxTime = 10 * time.Minute
	}

	// Parse Worker ID
	if workerID.Valid {
		job.WorkerID = &workerID.String
	}

	// Parse Timestamps
	job.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	job.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if startedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, startedAtStr.String)
		job.StartedAt = &t
	}

	if completedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, completedAtStr.String)
		job.CompletedAt = &t
	}

	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}

	job.Progress.CalculatePercentage()

	return job, nil
}

// List retrieves jobs with optional filtering
func (r *JobRepository) List(ctx context.Context, params domain.JobListParams) ([]*domain.Job, int, error) {
	var conditions []string
	var args []interface{}

	if params.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *params.Status)
	}

	if params.WorkerID != nil {
		conditions = append(conditions, "worker_id = ?")
		args = append(args, *params.WorkerID)
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
		LIMIT ? OFFSET ?
	`, whereClause, orderBy, orderDir)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job := &domain.Job{}
		var idStr, statusStr string
		var keywordsJSON, proxiesJSON string
		var maxTimeStr string
		var workerID sql.NullString
		var createdAtStr, updatedAtStr string
		var startedAtStr, completedAtStr sql.NullString
		var errorMessage sql.NullString

		err := rows.Scan(
			&idStr, &job.Name, &statusStr, &job.Priority,
			&keywordsJSON, &job.Config.Lang, &job.Config.GeoLat, &job.Config.GeoLon,
			&job.Config.Zoom, &job.Config.Radius, &job.Config.Depth,
			&job.Config.FastMode, &job.Config.ExtractEmail, &maxTimeStr, &proxiesJSON,
			&job.Progress.TotalPlaces, &job.Progress.ScrapedPlaces, &job.Progress.FailedPlaces,
			&workerID, &createdAtStr, &updatedAtStr, &startedAtStr, &completedAtStr,
			&errorMessage,
		)
		if err != nil {
			return nil, 0, err
		}

		job.ID, _ = uuid.Parse(idStr)
		job.Status = domain.JobStatus(statusStr)
		_ = json.Unmarshal([]byte(keywordsJSON), &job.Config.Keywords)
		if proxiesJSON != "" {
			_ = json.Unmarshal([]byte(proxiesJSON), &job.Config.Proxies)
		}
		job.Config.MaxTime, _ = time.ParseDuration(maxTimeStr)

		if workerID.Valid {
			job.WorkerID = &workerID.String
		}
		job.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		job.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		if startedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, startedAtStr.String)
			job.StartedAt = &t
		}
		if completedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, completedAtStr.String)
			job.CompletedAt = &t
		}
		if errorMessage.Valid {
			job.ErrorMessage = &errorMessage.String
		}

		job.Progress.CalculatePercentage()
		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	query := `
		UPDATE jobs_queue SET
			name = ?, status = ?, priority = ?,
			keywords = ?, lang = ?, geo_lat = ?, geo_lon = ?,
			zoom = ?, radius = ?, depth = ?,
			fast_mode = ?, extract_email = ?, max_time = ?, proxies = ?,
			total_places = ?, scraped_places = ?, failed_places = ?,
			worker_id = ?, started_at = ?, completed_at = ?,
			error_message = ?, updated_at = ?
		WHERE id = ?
	`

	keywordsJSON, _ := json.Marshal(job.Config.Keywords)
	proxiesJSON, _ := json.Marshal(job.Config.Proxies)

	var startedAtStr, completedAtStr interface{}
	if job.StartedAt != nil {
		startedAtStr = job.StartedAt.Format(time.RFC3339)
	}
	if job.CompletedAt != nil {
		completedAtStr = job.CompletedAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, query,
		job.Name, job.Status, job.Priority,
		string(keywordsJSON), job.Config.Lang, job.Config.GeoLat, job.Config.GeoLon,
		job.Config.Zoom, job.Config.Radius, job.Config.Depth,
		job.Config.FastMode, job.Config.ExtractEmail, job.Config.MaxTime.String(), string(proxiesJSON),
		job.Progress.TotalPlaces, job.Progress.ScrapedPlaces, job.Progress.FailedPlaces,
		job.WorkerID, startedAtStr, completedAtStr,
		job.ErrorMessage, time.Now().UTC().Format(time.RFC3339),
		job.ID.String(),
	)

	return err
}

// Delete deletes a job by ID
func (r *JobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM jobs_queue WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id.String())
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
	now := time.Now().UTC().Format(time.RFC3339)

	switch status {
	case domain.JobStatusRunning:
		query = `UPDATE jobs_queue SET status = ?, started_at = ?, updated_at = ? WHERE id = ?`
		_, err := r.db.ExecContext(ctx, query, status, now, now, id.String())
		return err
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled:
		query = `UPDATE jobs_queue SET status = ?, completed_at = ?, worker_id = NULL, updated_at = ? WHERE id = ?`
		_, err := r.db.ExecContext(ctx, query, status, now, now, id.String())
		return err
	default:
		query = `UPDATE jobs_queue SET status = ?, updated_at = ? WHERE id = ?`
		_, err := r.db.ExecContext(ctx, query, status, now, id.String())
		return err
	}
}

// UpdateProgress updates the progress of a job
func (r *JobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress domain.JobProgress) error {
	query := `
		UPDATE jobs_queue SET
			total_places = ?,
			scraped_places = ?,
			failed_places = ?,
			updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		progress.TotalPlaces, progress.ScrapedPlaces, progress.FailedPlaces, now, id.String())
	return err
}

// ClaimJob claims a pending job for a worker (atomic operation using transaction)
func (r *JobRepository) ClaimJob(ctx context.Context, workerID string) (*domain.Job, error) {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find the first pending job
	selectQuery := `
		SELECT id FROM jobs_queue
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
	`

	var jobIDStr string
	err = tx.QueryRowContext(ctx, selectQuery).Scan(&jobIDStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // No pending jobs
	}
	if err != nil {
		return nil, err
	}

	// Update the job
	updateQuery := `
		UPDATE jobs_queue SET
			status = 'running',
			worker_id = ?,
			started_at = ?,
			updated_at = ?
		WHERE id = ? AND status = 'pending'
	`
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := tx.ExecContext(ctx, updateQuery, workerID, now, now, jobIDStr)
	if err != nil {
		return nil, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// Job was claimed by another worker between select and update
		return nil, nil // Or return specific error
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	jobID, _ := uuid.Parse(jobIDStr)
	return r.GetByID(ctx, jobID)
}

// ReleaseJob releases a job back to pending status
func (r *JobRepository) ReleaseJob(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE jobs_queue SET
			status = 'pending',
			worker_id = NULL,
			started_at = NULL,
			updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query, now, id.String())
	return err
}

// GetStats retrieves job statistics
func (r *JobRepository) GetStats(ctx context.Context) (*domain.JobStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END) as queued,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running,
			SUM(CASE WHEN status = 'paused' THEN 1 ELSE 0 END) as paused,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
		FROM jobs_queue
	`

	stats := &domain.JobStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.Total, &stats.Pending, &stats.Queued, &stats.Running,
		&stats.Paused, &stats.Completed, &stats.Failed, &stats.Cancelled,
	)

	return stats, err
}
