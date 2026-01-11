package managerrunner

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gosom/google-maps-scraper/internal/api"
	"github.com/gosom/google-maps-scraper/internal/api/handlers"
	"github.com/gosom/google-maps-scraper/internal/domain"
	"github.com/gosom/google-maps-scraper/internal/heartbeat"
	"github.com/gosom/google-maps-scraper/internal/repository/postgres"
	"github.com/gosom/google-maps-scraper/internal/repository/sqlite"
	"github.com/gosom/google-maps-scraper/internal/service"
	"github.com/gosom/google-maps-scraper/runner"
	"golang.org/x/sync/errgroup"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds configuration for the manager runner
type Config struct {
	// DatabaseURL is the PostgreSQL connection string or SQLite file path
	DatabaseURL string

	// Address is the HTTP server address
	Address string

	// DataFolder is where to store temporary files
	DataFolder string
}

// ManagerRunner runs the manager (Web UI + API) without scraping
type ManagerRunner struct {
	cfg       *Config
	db        *sql.DB
	srv       *http.Server
	jobSvc    *service.JobService
	workerSvc *service.WorkerService
	resultSvc *service.ResultService
	statsSvc  *service.StatsService
	hbMonitor *heartbeat.Monitor
}

// New creates a new ManagerRunner
func New(cfg *Config) (runner.Runner, error) {
	// Default address
	if cfg.Address == "" {
		cfg.Address = ":8080"
	}

	// Default data folder
	if cfg.DataFolder == "" {
		cfg.DataFolder = "."
	}

	if err := os.MkdirAll(cfg.DataFolder, os.ModePerm); err != nil {
		return nil, err
	}

	var (
		db         *sql.DB
		jobRepo    domain.JobRepository
		workerRepo domain.WorkerRepository
		resultRepo domain.ResultRepository
		err        error
	)

	// Check if using PostgreSQL
	isPostgres := strings.HasPrefix(cfg.DatabaseURL, "postgres://") || strings.HasPrefix(cfg.DatabaseURL, "postgresql://")

	if isPostgres {
		// Open PostgreSQL connection
		db, err = postgres.OpenConnection(cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// Run migrations automatically
		if err := runEmbeddedMigrations(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		// Initialize repositories
		repos := postgres.NewRepositories(db)
		jobRepo = repos.Jobs
		workerRepo = repos.Workers
		resultRepo = repos.Results
	} else {
		// Default to SQLite
		if cfg.DatabaseURL == "" {
			cfg.DatabaseURL = "gmaps.db"
		}

		// Open SQLite connection
		db, err = sqlite.OpenConnection(cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// Run migrations automatically
		if err := sqlite.RunMigrations(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		// Initialize repositories
		repos := sqlite.NewRepositories(db)
		jobRepo = repos.Jobs
		workerRepo = repos.Workers
		resultRepo = repos.Results
	}

	// Initialize services
	jobSvc := service.NewJobService(jobRepo, resultRepo)
	workerSvc := service.NewWorkerService(workerRepo, jobRepo)
	resultSvc := service.NewResultService(resultRepo)
	statsSvc := service.NewStatsService(jobRepo, workerRepo, resultRepo)

	// Initialize handlers
	jobHandler := handlers.NewJobHandler(jobSvc, resultSvc)
	workerHandler := handlers.NewWorkerHandler(workerSvc)
	statsHandler := handlers.NewStatsHandler(statsSvc)

	// Setup router
	router := api.NewRouter(jobHandler, workerHandler, statsHandler)
	apiToken := os.Getenv("API_TOKEN")
	handler := router.Setup(apiToken)

	// Create HTTP server
	srv := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// Create heartbeat monitor
	hbMonitor := heartbeat.NewMonitor(workerSvc, 0)

	return &ManagerRunner{
		cfg:       cfg,
		db:        db,
		srv:       srv,
		jobSvc:    jobSvc,
		workerSvc: workerSvc,
		resultSvc: resultSvc,
		statsSvc:  statsSvc,
		hbMonitor: hbMonitor,
	}, nil
}

// Run starts the manager
func (m *ManagerRunner) Run(ctx context.Context) error {
	egroup, ctx := errgroup.WithContext(ctx)

	// Start heartbeat monitor
	egroup.Go(func() error {
		return m.hbMonitor.Run(ctx)
	})

	// Start HTTP server
	egroup.Go(func() error {
		return m.startServer(ctx)
	})

	return egroup.Wait()
}

// Close cleans up resources
func (m *ManagerRunner) Close(_ context.Context) error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *ManagerRunner) startServer(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := m.srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("error shutting down server: %v", err)
		}
	}()

	log.Printf("manager API server starting on http://localhost%s", m.cfg.Address)
	if strings.HasPrefix(m.cfg.DatabaseURL, "postgres") {
		log.Printf("using PostgreSQL database")
	} else {
		log.Printf("using SQLite database: %s", m.cfg.DatabaseURL)
	}
	log.Printf("API endpoints available at /api/v2/")

	err := m.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// runEmbeddedMigrations runs migrations from embedded files (for PostgreSQL)
func runEmbeddedMigrations(db *sql.DB) error {
	// Create migrations tracking table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read embedded migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort migration files by name
	var migrations []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	// Run each migration
	for _, migration := range migrations {
		version := strings.TrimSuffix(migration, ".up.sql")

		// Check if already applied
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", version, err)
		}

		if exists {
			continue
		}

		// Read and execute migration
		content, err := migrationsFS.ReadFile("migrations/" + migration)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		log.Printf("applying migration: %s", migration)

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration, err)
		}

		// Record migration
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
	}

	log.Println("database migrations completed")
	return nil
}
