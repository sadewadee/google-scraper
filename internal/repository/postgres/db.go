package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenConnection opens a PostgreSQL connection
func OpenConnection(dsn string) (*sql.DB, error) {
	// Try to parse and re-encode the DSN to handle special characters in password
	parsedDSN, err := sanitizeDSN(dsn)
	if err != nil {
		// If parsing fails, try using the original DSN
		parsedDSN = dsn
	}

	db, err := sql.Open("pgx", parsedDSN)
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

// sanitizeDSN attempts to parse and properly encode the DSN
func sanitizeDSN(dsn string) (string, error) {
	// Check if it's a URL format (postgres:// or postgresql://)
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		// Assume it's already in key-value format, return as-is
		return dsn, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	// Re-encode the password if present
	if u.User != nil {
		password, hasPassword := u.User.Password()
		if hasPassword {
			// The password needs to be properly URL-encoded
			u.User = url.UserPassword(u.User.Username(), password)
		}
	}

	return u.String(), nil
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
