package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenConnection opens a PostgreSQL connection
func OpenConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// Repositories holds all repository instances
type Repositories struct {
	Jobs    *JobRepository
	Workers *WorkerRepository
	Results *ResultRepository
}

// NewRepositories creates all repositories
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Jobs:    NewJobRepository(db),
		Workers: NewWorkerRepository(db),
		Results: NewResultRepository(db),
	}
}
