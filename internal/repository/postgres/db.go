package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenConnection opens a PostgreSQL connection
func OpenConnection(dsn string) (*sql.DB, error) {
	log.Printf("[DB] Opening PostgreSQL connection...")

	// Try to parse and re-encode the DSN to handle special characters in password
	parsedDSN, err := sanitizeDSN(dsn)
	if err != nil {
		// If parsing fails, try using the original DSN
		parsedDSN = dsn
	}

	db, err := sql.Open("pgx", parsedDSN)
	if err != nil {
		log.Printf("[DB] Failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	log.Printf("[DB] Pinging database...")
	pingStart := time.Now()
	if err := db.Ping(); err != nil {
		log.Printf("[DB] Ping failed after %v: %v", time.Since(pingStart), err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Printf("[DB] Ping successful in %v", time.Since(pingStart))

	// Connection pool settings
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Printf("[DB] Connection pool configured: maxOpen=100, maxIdle=25, maxLifetime=5m")

	return db, nil
}

// sanitizeDSN converts URL format DSN to key-value format to handle special characters in password
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

	// Convert to key-value format which handles special characters better
	var parts []string

	// Host and port
	host := u.Hostname()
	port := u.Port()
	if host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", host))
	}
	if port != "" {
		parts = append(parts, fmt.Sprintf("port=%s", port))
	}

	// Database name (from path, remove leading slash)
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", dbname))
	}

	// User and password
	if u.User != nil {
		username := u.User.Username()
		if username != "" {
			parts = append(parts, fmt.Sprintf("user=%s", username))
		}
		password, hasPassword := u.User.Password()
		if hasPassword {
			// Escape single quotes in password by doubling them
			password = strings.ReplaceAll(password, "'", "''")
			parts = append(parts, fmt.Sprintf("password='%s'", password))
		}
	}

	// Query parameters (like sslmode)
	for key, values := range u.Query() {
		if len(values) > 0 {
			parts = append(parts, fmt.Sprintf("%s=%s", key, values[0]))
		}
	}

	return strings.Join(parts, " "), nil
}

// Repositories holds all repository instances
type Repositories struct {
	Jobs    *JobRepository
	Workers *WorkerRepository
	Results *ResultRepository
	Proxies *ProxyRepository
}

// NewRepositories creates all repositories
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Jobs:    NewJobRepository(db),
		Workers: NewWorkerRepository(db),
		Results: NewResultRepository(db),
		Proxies: NewProxyRepository(db),
	}
}
