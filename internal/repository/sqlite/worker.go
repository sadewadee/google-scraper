package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sadewadee/google-scraper/internal/domain"
)

// WorkerRepository implements domain.WorkerRepository for SQLite
type WorkerRepository struct {
	db *sql.DB
}

// NewWorkerRepository creates a new WorkerRepository
func NewWorkerRepository(db *sql.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

// Upsert creates or updates a worker (for heartbeat)
func (r *WorkerRepository) Upsert(ctx context.Context, worker *domain.Worker) error {
	query := `
		INSERT INTO workers (
			id, hostname, status, current_job_id,
			jobs_completed, places_scraped, last_heartbeat, created_at
		) VALUES (?, ?, ?, ?, 0, 0, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			hostname = excluded.hostname,
			status = excluded.status,
			current_job_id = excluded.current_job_id,
			last_heartbeat = excluded.last_heartbeat
	`

	jobID := sql.NullString{}
	if worker.CurrentJobID != nil {
		jobID.String = worker.CurrentJobID.String()
		jobID.Valid = true
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		worker.ID, worker.Hostname, worker.Status, jobID,
		now, now,
	)

	return err
}

// GetByID retrieves a worker by ID
func (r *WorkerRepository) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	query := `
		SELECT
			w.id, w.hostname, w.status, w.current_job_id,
			w.jobs_completed, w.places_scraped, w.last_heartbeat, w.created_at,
			j.name
		FROM workers w
		LEFT JOIN jobs_queue j ON w.current_job_id = j.id
		WHERE w.id = ?
	`

	worker := &domain.Worker{}
	var currentJobID sql.NullString
	var currentJobName sql.NullString
	var lastHeartbeatStr, createdAtStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&worker.ID, &worker.Hostname, &worker.Status, &currentJobID,
		&worker.JobsCompleted, &worker.PlacesScraped, &lastHeartbeatStr, &createdAtStr,
		&currentJobName,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if currentJobID.Valid {
		uid, err := uuid.Parse(currentJobID.String)
		if err == nil {
			worker.CurrentJobID = &uid
		}
	}
	if currentJobName.Valid {
		worker.CurrentJobName = &currentJobName.String
	}

	worker.LastHeartbeat, _ = time.Parse(time.RFC3339, lastHeartbeatStr)
	worker.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	return worker, nil
}

// List retrieves all workers
func (r *WorkerRepository) List(ctx context.Context, params domain.WorkerListParams) ([]*domain.Worker, error) {
	query := `
		SELECT
			w.id, w.hostname, w.status, w.current_job_id,
			w.jobs_completed, w.places_scraped, w.last_heartbeat, w.created_at,
			j.name
		FROM workers w
		LEFT JOIN jobs_queue j ON w.current_job_id = j.id
		ORDER BY w.last_heartbeat DESC
	`

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}
	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", params.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		worker := &domain.Worker{}
		var currentJobID sql.NullString
		var currentJobName sql.NullString
		var lastHeartbeatStr, createdAtStr string

		err := rows.Scan(
			&worker.ID, &worker.Hostname, &worker.Status, &currentJobID,
			&worker.JobsCompleted, &worker.PlacesScraped, &lastHeartbeatStr, &createdAtStr,
			&currentJobName,
		)
		if err != nil {
			return nil, err
		}

		if currentJobID.Valid {
			uid, err := uuid.Parse(currentJobID.String)
			if err == nil {
				worker.CurrentJobID = &uid
			}
		}
		if currentJobName.Valid {
			worker.CurrentJobName = &currentJobName.String
		}

		worker.LastHeartbeat, _ = time.Parse(time.RFC3339, lastHeartbeatStr)
		worker.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		workers = append(workers, worker)
	}

	return workers, rows.Err()
}

// Delete deletes a worker by ID
func (r *WorkerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workers WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdateStatus updates only the status of a worker
func (r *WorkerRepository) UpdateStatus(ctx context.Context, id string, status domain.WorkerStatus) error {
	query := `UPDATE workers SET status = ?, last_heartbeat = ? WHERE id = ?`
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, query, status, now, id)
	return err
}

// MarkOfflineWorkers marks workers as offline if heartbeat is stale
func (r *WorkerRepository) MarkOfflineWorkers(ctx context.Context, timeout int) (int, error) {
	query := `
		UPDATE workers
		SET status = 'offline'
		WHERE status != 'offline'
		AND datetime(last_heartbeat) < datetime('now', '-' || ? || ' seconds')
	`
	// Note: timeout is int seconds.
	// SQLite datetime modifiers: '-30 seconds'

	result, err := r.db.ExecContext(ctx, query, fmt.Sprintf("%d", timeout))
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// GetStats retrieves worker statistics
func (r *WorkerRepository) GetStats(ctx context.Context) (*domain.WorkerStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status != 'offline' THEN 1 ELSE 0 END) as online,
			SUM(CASE WHEN status = 'busy' THEN 1 ELSE 0 END) as busy,
			SUM(CASE WHEN status = 'idle' THEN 1 ELSE 0 END) as idle
		FROM workers
	`

	stats := &domain.WorkerStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalWorkers, &stats.OnlineWorkers, &stats.BusyWorkers, &stats.IdleWorkers,
	)

	return stats, err
}

// IncrementStats increments worker statistics
func (r *WorkerRepository) IncrementStats(ctx context.Context, id string, jobsCompleted, placesScraped int) error {
	query := `
		UPDATE workers SET
			jobs_completed = jobs_completed + ?,
			places_scraped = places_scraped + ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, jobsCompleted, placesScraped, id)
	return err
}
