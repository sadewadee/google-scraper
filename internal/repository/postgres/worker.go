package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// WorkerRepository implements domain.WorkerRepository for PostgreSQL
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
		INSERT INTO workers (id, hostname, status, current_job_id, last_heartbeat, created_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			status = EXCLUDED.status,
			current_job_id = EXCLUDED.current_job_id,
			last_heartbeat = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		worker.ID, worker.Hostname, worker.Status, worker.CurrentJobID)
	return err
}

// GetByID retrieves a worker by ID
func (r *WorkerRepository) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	query := `
		SELECT
			w.id, w.hostname, w.status, w.current_job_id,
			w.jobs_completed, w.places_scraped, w.last_heartbeat, w.created_at,
			j.name as job_name
		FROM workers w
		LEFT JOIN jobs_queue j ON w.current_job_id = j.id
		WHERE w.id = $1
	`

	worker := &domain.Worker{}
	var currentJobID *uuid.UUID

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&worker.ID, &worker.Hostname, &worker.Status, &currentJobID,
		&worker.JobsCompleted, &worker.PlacesScraped, &worker.LastHeartbeat, &worker.CreatedAt,
		&worker.CurrentJobName,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	worker.CurrentJobID = currentJobID
	return worker, nil
}

// List retrieves all workers
func (r *WorkerRepository) List(ctx context.Context, params domain.WorkerListParams) ([]*domain.Worker, error) {
	query := `
		SELECT
			w.id, w.hostname, w.status, w.current_job_id,
			w.jobs_completed, w.places_scraped, w.last_heartbeat, w.created_at,
			j.name as job_name
		FROM workers w
		LEFT JOIN jobs_queue j ON w.current_job_id = j.id
		ORDER BY w.last_heartbeat DESC
	`

	// Apply limit if provided
	if params.Limit > 0 {
		query += " LIMIT $1 OFFSET $2"
	}

	var rows *sql.Rows
	var err error

	if params.Limit > 0 {
		rows, err = r.db.QueryContext(ctx, query, params.Limit, params.Offset)
	} else {
		rows, err = r.db.QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		worker := &domain.Worker{}
		var currentJobID *uuid.UUID

		err := rows.Scan(
			&worker.ID, &worker.Hostname, &worker.Status, &currentJobID,
			&worker.JobsCompleted, &worker.PlacesScraped, &worker.LastHeartbeat, &worker.CreatedAt,
			&worker.CurrentJobName,
		)
		if err != nil {
			return nil, err
		}

		worker.CurrentJobID = currentJobID

		// Update status based on heartbeat
		if !worker.IsOnline(domain.HeartbeatTimeout) {
			worker.Status = domain.WorkerStatusOffline
		}

		workers = append(workers, worker)
	}

	return workers, rows.Err()
}

// Delete deletes a worker by ID
func (r *WorkerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workers WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdateStatus updates only the status of a worker
func (r *WorkerRepository) UpdateStatus(ctx context.Context, id string, status domain.WorkerStatus) error {
	query := `UPDATE workers SET status = $2, last_heartbeat = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

// MarkOfflineWorkers marks workers as offline if heartbeat is stale
func (r *WorkerRepository) MarkOfflineWorkers(ctx context.Context, timeoutSeconds int) (int, error) {
	query := `
		UPDATE workers SET
			status = 'offline',
			current_job_id = NULL
		WHERE last_heartbeat < NOW() - INTERVAL '1 second' * $1
		AND status != 'offline'
	`

	result, err := r.db.ExecContext(ctx, query, timeoutSeconds)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// GetStats retrieves worker statistics
func (r *WorkerRepository) GetStats(ctx context.Context) (*domain.WorkerStats, error) {
	// Use heartbeat timeout to determine online status
	timeout := int(domain.HeartbeatTimeout.Seconds())

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE last_heartbeat >= NOW() - INTERVAL '1 second' * $1) as online,
			COUNT(*) FILTER (WHERE status = 'busy' AND last_heartbeat >= NOW() - INTERVAL '1 second' * $1) as busy,
			COUNT(*) FILTER (WHERE status = 'idle' AND last_heartbeat >= NOW() - INTERVAL '1 second' * $1) as idle
		FROM workers
	`

	stats := &domain.WorkerStats{}
	err := r.db.QueryRowContext(ctx, query, timeout).Scan(
		&stats.TotalWorkers, &stats.OnlineWorkers, &stats.BusyWorkers, &stats.IdleWorkers,
	)

	return stats, err
}

// IncrementStats increments worker statistics
func (r *WorkerRepository) IncrementStats(ctx context.Context, id string, jobsCompleted, placesScraped int) error {
	query := `
		UPDATE workers SET
			jobs_completed = jobs_completed + $2,
			places_scraped = places_scraped + $3,
			last_heartbeat = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, jobsCompleted, placesScraped)
	return err
}

// CleanupStaleWorkers removes workers that haven't sent heartbeat in a long time
func (r *WorkerRepository) CleanupStaleWorkers(ctx context.Context, maxAge time.Duration) (int, error) {
	query := `
		DELETE FROM workers
		WHERE last_heartbeat < NOW() - INTERVAL '1 second' * $1
	`

	result, err := r.db.ExecContext(ctx, query, int(maxAge.Seconds()))
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}
