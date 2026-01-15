package migration

import (
	"context"
	"database/sql"
)

// MigrationState represents the current state of the database
type MigrationState int

const (
	StateFreshInstall      MigrationState = iota // No tables exist
	StateBothExistUnlinked                       // Both tables, no parent_job_id
	StateOnlyGmapsJobs                           // Only gmaps_jobs exists
	StateOnlyJobsQueue                           // Only jobs_queue exists
	StateAlreadyMigrated                         // parent_job_id column exists
)

// String returns a human-readable name for the migration state
func (s MigrationState) String() string {
	switch s {
	case StateFreshInstall:
		return "FreshInstall"
	case StateBothExistUnlinked:
		return "BothExistUnlinked"
	case StateOnlyGmapsJobs:
		return "OnlyGmapsJobs"
	case StateOnlyJobsQueue:
		return "OnlyJobsQueue"
	case StateAlreadyMigrated:
		return "AlreadyMigrated"
	default:
		return "Unknown"
	}
}

// DetectMigrationState checks current database state
func DetectMigrationState(ctx context.Context, db *sql.DB) (MigrationState, error) {
	var jobsQueueExists, gmapsJobsExists, parentJobIdExists bool

	// Check if jobs_queue exists
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'jobs_queue'
		)
	`).Scan(&jobsQueueExists)
	if err != nil {
		return 0, err
	}

	// Check if gmaps_jobs exists
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'gmaps_jobs'
		)
	`).Scan(&gmapsJobsExists)
	if err != nil {
		return 0, err
	}

	// Check if parent_job_id column exists in gmaps_jobs
	if gmapsJobsExists {
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.columns
				WHERE table_name = 'gmaps_jobs' AND column_name = 'parent_job_id'
			)
		`).Scan(&parentJobIdExists)
		if err != nil {
			return 0, err
		}
	}

	// Determine state
	if parentJobIdExists {
		return StateAlreadyMigrated, nil
	}
	if !jobsQueueExists && !gmapsJobsExists {
		return StateFreshInstall, nil
	}
	if jobsQueueExists && gmapsJobsExists {
		return StateBothExistUnlinked, nil
	}
	if gmapsJobsExists && !jobsQueueExists {
		return StateOnlyGmapsJobs, nil
	}
	if jobsQueueExists && !gmapsJobsExists {
		return StateOnlyJobsQueue, nil
	}

	return StateFreshInstall, nil
}
