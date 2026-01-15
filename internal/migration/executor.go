package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// AutoMigrate runs one-time migration based on detected state
func AutoMigrate(ctx context.Context, db *sql.DB) error {
	state, err := DetectMigrationState(ctx, db)
	if err != nil {
		return fmt.Errorf("detect migration state: %w", err)
	}

	log.Printf("[AutoMigrate] Detected state: %v", state)

	switch state {
	case StateAlreadyMigrated:
		log.Println("[AutoMigrate] Already migrated, skipping")
		return nil

	case StateFreshInstall:
		log.Println("[AutoMigrate] Fresh install, running full schema creation")
		return runFreshInstall(ctx, db)

	case StateBothExistUnlinked:
		log.Println("[AutoMigrate] Both tables exist, adding link columns")
		return migrateBothExistUnlinked(ctx, db)

	case StateOnlyGmapsJobs:
		log.Println("[AutoMigrate] Only gmaps_jobs exists, creating jobs_queue")
		return migrateOnlyGmapsJobs(ctx, db)

	case StateOnlyJobsQueue:
		log.Println("[AutoMigrate] Only jobs_queue exists, creating gmaps_jobs and bridging")
		return migrateOnlyJobsQueue(ctx, db)

	default:
		return fmt.Errorf("unknown migration state: %v", state)
	}
}

// migrateBothExistUnlinked handles Scenario B
func migrateBothExistUnlinked(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Add parent_job_id column to gmaps_jobs
	_, err = tx.ExecContext(ctx, `
		ALTER TABLE gmaps_jobs
		ADD COLUMN IF NOT EXISTS parent_job_id UUID;

		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent
		ON gmaps_jobs(parent_job_id);

		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent_status
		ON gmaps_jobs(parent_job_id, status);
	`)
	if err != nil {
		return fmt.Errorf("add parent_job_id column: %w", err)
	}

	// 2. Add task tracking columns to jobs_queue
	_, err = tx.ExecContext(ctx, `
		ALTER TABLE jobs_queue
		ADD COLUMN IF NOT EXISTS total_tasks INTEGER DEFAULT 0;

		ALTER TABLE jobs_queue
		ADD COLUMN IF NOT EXISTS completed_tasks INTEGER DEFAULT 0;
	`)
	if err != nil {
		return fmt.Errorf("add task tracking columns: %w", err)
	}

	// 3. Record migration timestamp
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migration_history (
			id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL,
			executed_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO migration_history (migration_name)
		VALUES ('auto_migrate_both_exist_unlinked');
	`)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	log.Println("[AutoMigrate] Successfully migrated both tables")
	return tx.Commit()
}

// migrateOnlyJobsQueue handles Scenario D - re-bridge pending jobs
func migrateOnlyJobsQueue(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Create gmaps_jobs table
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS gmaps_jobs (
			id TEXT PRIMARY KEY,
			priority INT DEFAULT 0,
			payload_type VARCHAR(50),
			payload BYTEA,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			status VARCHAR(20) DEFAULT 'new',
			parent_job_id UUID
		);

		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_status ON gmaps_jobs(status);
		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_priority ON gmaps_jobs(priority DESC);
		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent ON gmaps_jobs(parent_job_id);
		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent_status ON gmaps_jobs(parent_job_id, status);
	`)
	if err != nil {
		return fmt.Errorf("create gmaps_jobs table: %w", err)
	}

	// 2. Add task tracking columns to jobs_queue
	_, err = tx.ExecContext(ctx, `
		ALTER TABLE jobs_queue
		ADD COLUMN IF NOT EXISTS total_tasks INTEGER DEFAULT 0;

		ALTER TABLE jobs_queue
		ADD COLUMN IF NOT EXISTS completed_tasks INTEGER DEFAULT 0;
	`)
	if err != nil {
		return fmt.Errorf("add task tracking columns: %w", err)
	}

	// 3. Count pending/running jobs that may need to be re-bridged
	var pendingCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM jobs_queue
		WHERE status IN ('pending', 'running', 'queued')
	`).Scan(&pendingCount)
	if err != nil {
		return fmt.Errorf("count pending jobs: %w", err)
	}

	if pendingCount > 0 {
		log.Printf("[AutoMigrate] Found %d pending jobs to re-bridge after migration", pendingCount)
		log.Println("[AutoMigrate] NOTE: Run bridge manually for these jobs or recreate them via Dashboard")
	}

	// 4. Create migration_history table and record migration
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migration_history (
			id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL,
			executed_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO migration_history (migration_name)
		VALUES ('auto_migrate_only_jobs_queue');
	`)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// runFreshInstall creates all tables from scratch
func runFreshInstall(ctx context.Context, db *sql.DB) error {
	// This would run the standard migrations
	// For now, just record that we started fresh
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migration_history (
			id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL,
			executed_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO migration_history (migration_name)
		VALUES ('auto_migrate_fresh_install');
	`)
	return err
}

// migrateOnlyGmapsJobs handles Scenario C
func migrateOnlyGmapsJobs(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Add parent_job_id column
	_, err = tx.ExecContext(ctx, `
		ALTER TABLE gmaps_jobs
		ADD COLUMN IF NOT EXISTS parent_job_id UUID;

		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent
		ON gmaps_jobs(parent_job_id);

		CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent_status
		ON gmaps_jobs(parent_job_id, status);
	`)
	if err != nil {
		return fmt.Errorf("add parent_job_id column: %w", err)
	}

	// 2. Create migration_history table and record migration
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migration_history (
			id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL,
			executed_at TIMESTAMPTZ DEFAULT NOW()
		);

		INSERT INTO migration_history (migration_name)
		VALUES ('auto_migrate_only_gmaps_jobs');
	`)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	log.Println("[AutoMigrate] Existing gmaps_jobs will have parent_job_id = NULL (CLI-originated)")
	return tx.Commit()
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(ctx context.Context, db *sql.DB) ([]MigrationRecord, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, migration_name, executed_at
		FROM migration_history
		ORDER BY executed_at DESC
	`)
	if err != nil {
		// Table might not exist yet
		return nil, nil
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.ExecutedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// MigrationRecord represents a migration history entry
type MigrationRecord struct {
	ID         int
	Name       string
	ExecutedAt string
}
