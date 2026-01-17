package managerrunner

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sadewadee/google-scraper/internal/api"
	"github.com/sadewadee/google-scraper/internal/api/handlers"
	"github.com/sadewadee/google-scraper/internal/cache"
	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/sadewadee/google-scraper/internal/heartbeat"
	"github.com/sadewadee/google-scraper/internal/migration"
	"github.com/sadewadee/google-scraper/internal/mq"
	"github.com/sadewadee/google-scraper/internal/proxygate"
	"github.com/sadewadee/google-scraper/internal/queue"
	"github.com/sadewadee/google-scraper/internal/repository/postgres"
	"github.com/sadewadee/google-scraper/internal/repository/sqlite"
	"github.com/sadewadee/google-scraper/internal/service"
	"github.com/sadewadee/google-scraper/internal/spawner"
	gmapspostgres "github.com/sadewadee/google-scraper/postgres"
	"github.com/sadewadee/google-scraper/runner"
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

	// StaticFolder is the path to static frontend files
	StaticFolder string

	// DataFolder is where to store temporary files
	DataFolder string

	// Redis configuration for job queue and cache
	RedisURL  string
	RedisAddr string
	RedisPass string
	RedisDB   int

	// RabbitMQ configuration for job queue
	RabbitMQURL string

	// Spawner configuration for auto-spawning workers
	SpawnerType        string            // none, docker, swarm, lambda
	SpawnerImage       string            // Docker image for worker containers
	SpawnerNetwork     string            // Docker network to attach workers
	SpawnerConcurrency int               // Concurrency per spawned worker
	SpawnerMaxWorkers  int               // Max concurrent workers (0 = unlimited)
	SpawnerAutoRemove  bool              // Auto-remove containers after exit
	SpawnerLabels      map[string]string // Labels for spawned containers
	SpawnerConstraints []string          // Swarm placement constraints
	SpawnerManagerURL  string            // Manager URL for spawned workers (Dokploy: use service name)
	SpawnerProxies     string            // Proxy URL for spawned workers (e.g., socks5://manager:8081)

	// AWS Lambda spawner configuration
	SpawnerLambdaFunction   string // Lambda function name/ARN
	SpawnerLambdaRegion     string // AWS region for Lambda
	SpawnerLambdaInvocation string // Event (async) or RequestResponse (sync)
	SpawnerLambdaMaxConc    int    // Max concurrent Lambda invocations
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
	proxyGate *proxygate.ProxyGate
	jobQueue  *queue.Queue
	mqPub     mq.Publisher
	cache     cache.Cache
	spawner   spawner.Spawner
}

// New creates a new ManagerRunner
func New(cfg *Config, pg *proxygate.ProxyGate) (runner.Runner, error) {
	log.Println("manager: starting initialization...")

	// Default address
	if cfg.Address == "" {
		cfg.Address = ":8080"
	}

	// Default data folder
	if cfg.DataFolder == "" {
		cfg.DataFolder = "."
	}

	log.Printf("manager: address=%s, dataFolder=%s", cfg.Address, cfg.DataFolder)

	if err := os.MkdirAll(cfg.DataFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create data folder: %w", err)
	}

	var (
		db         *sql.DB
		jobRepo    domain.JobRepository
		workerRepo domain.WorkerRepository
		resultRepo domain.ResultRepository
		proxyRepo  domain.ProxyRepository
		businessListingRepo domain.BusinessListingRepository
		err        error
	)

	// Check if using PostgreSQL
	isPostgres := strings.HasPrefix(cfg.DatabaseURL, "postgres://") || strings.HasPrefix(cfg.DatabaseURL, "postgresql://")

	log.Printf("manager: database type isPostgres=%v", isPostgres)

	if isPostgres {
		log.Println("manager: connecting to PostgreSQL...")

		// Open PostgreSQL connection
		db, err = postgres.OpenConnection(cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		log.Println("manager: database connected, running migrations...")

		// Run migrations automatically
		if err := runEmbeddedMigrations(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		// Run auto-migration for Dashboard → DSN bridge
		log.Println("manager: running auto-migration for DSN bridge...")
		if err := migration.AutoMigrate(context.Background(), db); err != nil {
			db.Close()
			return nil, fmt.Errorf("auto-migrate failed: %w", err)
		}

		log.Println("manager: migrations completed, initializing repositories...")

		// Initialize repositories
		repos := postgres.NewRepositories(db)
		jobRepo = repos.Jobs
		workerRepo = repos.Workers
		resultRepo = repos.Results
		proxyRepo = repos.Proxies

		// Initialize BusinessListingRepository for normalized data access
		businessListingRepo = postgres.NewBusinessListingRepository(db)
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

	// Initialize Redis queue (optional - gracefully handles missing Redis)
	var jobQueue *queue.Queue
	var redisCache cache.Cache
	if cfg.RedisURL != "" || cfg.RedisAddr != "" {
		queueCfg := &queue.Config{
			RedisURL:  cfg.RedisURL,
			RedisAddr: cfg.RedisAddr,
			Password:  cfg.RedisPass,
			DB:        cfg.RedisDB,
		}
		q, err := queue.New(queueCfg)
		if err != nil {
			log.Printf("manager: WARNING - failed to connect to Redis queue: %v", err)
			log.Println("manager: continuing without Redis queue (workers must poll)")
		} else {
			jobQueue = q
			log.Println("manager: Redis queue connected")
		}

		// Initialize Redis cache (uses same Redis instance)
		cacheCfg := cache.Config{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPass,
			DB:       cfg.RedisDB,
		}
		// If RedisURL is specified but not Addr, parse the URL
		if cfg.RedisAddr == "" && cfg.RedisURL != "" {
			// Parse redis://user:pass@host:port/db format
			// For now, just log and skip (RedisAddr should be preferred)
			log.Printf("manager: Redis URL format not supported for cache, use -redis-addr")
		}
		if cfg.RedisAddr != "" {
			c, err := cache.NewRedisCache(cacheCfg)
			if err != nil {
				log.Printf("manager: WARNING - failed to connect to Redis cache: %v", err)
				log.Println("manager: continuing without caching (slower dashboard)")
				redisCache = cache.NewNoOpCache()
			} else {
				redisCache = c
				log.Println("manager: Redis cache connected")
			}
		} else {
			log.Println("manager: Redis cache not configured (no address)")
			redisCache = cache.NewNoOpCache()
		}
	} else {
		log.Println("manager: no Redis configured, workers will use HTTP polling")
		log.Println("manager: no Redis configured, using no-op cache (slower dashboard)")
		redisCache = cache.NewNoOpCache()
	}

	// Initialize RabbitMQ publisher (optional)
	var mqPublisher mq.Publisher
	if cfg.RabbitMQURL != "" {
		pub, err := mq.NewPublisher(mq.Config{URL: cfg.RabbitMQURL})
		if err != nil {
			log.Printf("manager: WARNING - failed to connect to RabbitMQ: %v", err)
			log.Println("manager: continuing without RabbitMQ (fallback to Redis queue if available)")
		} else {
			mqPublisher = pub
			log.Println("manager: RabbitMQ publisher connected")
		}
	} else {
		log.Println("manager: no RabbitMQ configured")
	}

	// Initialize services
	var jobSvc *service.JobService
	if isPostgres {
		// Use bridge to gmaps_jobs for DSN workers
		gmapsPusher := gmapspostgres.NewGmapsJobPusher(db)
		if mqPublisher != nil {
			// Use RabbitMQ publisher (preferred)
			jobSvc = service.NewJobServiceWithMQ(jobRepo, resultRepo, mqPublisher, gmapsPusher)
			log.Println("manager: JobService initialized with RabbitMQ + DSN bridge")
		} else {
			// Fallback to Redis queue
			jobSvc = service.NewJobServiceWithBridge(jobRepo, resultRepo, jobQueue, gmapsPusher)
			log.Println("manager: JobService initialized with Redis queue + DSN bridge")
		}
	} else {
		// SQLite mode - no bridge (deprecated mode)
		jobSvc = service.NewJobService(jobRepo, resultRepo, jobQueue)
		log.Println("manager: WARNING - using deprecated SQLite mode without DSN bridge")
	}
	workerSvc := service.NewWorkerService(workerRepo, jobRepo)
	resultSvc := service.NewResultService(resultRepo)
	statsSvc := service.NewStatsService(jobRepo, workerRepo, resultRepo)

	// Initialize spawner for auto-spawning workers
	var workerSpawner spawner.Spawner
	if cfg.SpawnerType != "" && cfg.SpawnerType != "none" {
		// Determine Manager URL for spawned workers
		// For Dokploy/Swarm: use service name (e.g., http://manager:8080)
		// For local Docker: use host.docker.internal or localhost
		managerURL := cfg.SpawnerManagerURL
		if managerURL == "" {
			// Auto-detect based on spawner type
			if cfg.SpawnerType == "swarm" {
				// For Swarm, try to use hostname as service name
				hostname, _ := os.Hostname()
				managerURL = fmt.Sprintf("http://%s%s", hostname, cfg.Address)
				log.Printf("manager: auto-detected manager URL for Swarm: %s", managerURL)
			} else {
				// For local Docker, use localhost
				managerURL = "http://localhost" + cfg.Address
			}
		}

		spawnerCfg := &spawner.Config{
			Type:        spawner.SpawnerType(cfg.SpawnerType),
			ManagerURL:  managerURL,
			RabbitMQURL: cfg.RabbitMQURL,
			RedisAddr:   cfg.RedisAddr,
			Proxies:     cfg.SpawnerProxies,
			Docker: spawner.DockerConfig{
				Image:       cfg.SpawnerImage,
				Network:     cfg.SpawnerNetwork,
				Concurrency: cfg.SpawnerConcurrency,
				AutoRemove:  cfg.SpawnerAutoRemove,
				MaxWorkers:  cfg.SpawnerMaxWorkers,
				Environment: cfg.SpawnerLabels,
			},
			Swarm: spawner.SwarmConfig{
				Image:       cfg.SpawnerImage,
				Network:     cfg.SpawnerNetwork,
				Concurrency: cfg.SpawnerConcurrency,
				MaxServices: cfg.SpawnerMaxWorkers,
				Labels:      cfg.SpawnerLabels,
				Constraints: cfg.SpawnerConstraints,
			},
			Lambda: spawner.LambdaConfig{
				FunctionName:   cfg.SpawnerLambdaFunction,
				Region:         cfg.SpawnerLambdaRegion,
				InvocationType: cfg.SpawnerLambdaInvocation,
				MaxConcurrent:  cfg.SpawnerLambdaMaxConc,
			},
		}

		sp, err := spawner.New(spawnerCfg)
		if err != nil {
			log.Printf("manager: WARNING - failed to initialize spawner: %v", err)
			log.Println("manager: continuing without auto-spawn (workers must be started manually)")
		} else {
			workerSpawner = sp
			jobSvc.SetSpawner(sp)
			log.Printf("manager: spawner initialized (type: %s)", cfg.SpawnerType)
		}
	} else {
		log.Println("manager: auto-spawn disabled (use -spawner docker|swarm|lambda to enable)")
	}

	log.Println("manager: services initialized, setting up router...")

	// Initialize handlers with caching
	// Use cached handlers for read operations to reduce database load
	var jobHandler *handlers.JobHandler
	var statsHandler *handlers.StatsHandler
	var resultHandler *handlers.ResultHandler

	// Cached handlers for read operations
	var cachedJobHandler *handlers.CachedJobHandler
	var cachedStatsHandler *handlers.CachedStatsHandler
	var cachedResultHandler *handlers.CachedResultHandler

	// Check if we have a real cache (not no-op)
	_, isNoOpCache := redisCache.(*cache.NoOpCache)
	if isNoOpCache {
		// No caching - use standard handlers
		jobHandler = handlers.NewJobHandler(jobSvc, resultSvc)
		statsHandler = handlers.NewStatsHandler(statsSvc)
		resultHandler = handlers.NewResultHandler(resultSvc)
		log.Println("manager: using standard handlers (no cache)")
	} else {
		// Use standard handlers for write operations
		jobHandler = handlers.NewJobHandlerWithCache(jobSvc, resultSvc, redisCache)
		statsHandler = handlers.NewStatsHandler(statsSvc)
		resultHandler = handlers.NewResultHandler(resultSvc)

		// Create cached handlers for read operations
		cachedJobHandler = handlers.NewCachedJobHandler(jobSvc, resultSvc, redisCache)
		cachedStatsHandler = handlers.NewCachedStatsHandler(statsSvc, redisCache)
		cachedResultHandler = handlers.NewCachedResultHandler(resultSvc, redisCache)
		log.Println("manager: using cached handlers for dashboard read operations")
	}
	workerHandler := handlers.NewWorkerHandler(workerSvc)
	proxyHandler := handlers.NewProxyHandler(pg, proxyRepo)

	// Create BusinessListingHandler for normalized data access (PostgreSQL only)
	var businessListingHandler *handlers.BusinessListingHandler
	if businessListingRepo != nil {
		businessListingSvc := service.NewBusinessListingService(businessListingRepo)
		businessListingHandler = handlers.NewBusinessListingHandler(businessListingSvc)
		log.Println("manager: BusinessListingHandler initialized for normalized data access")
	}

	// Create ProxyListRepository for listing individual proxies (PostgreSQL only)
	var proxyListRepo domain.ProxyListRepository
	if isPostgres {
		proxyListRepo = postgres.NewProxyListRepository(db)
		proxyHandler.SetProxyListRepo(proxyListRepo)
		// Also set pool repo for ProxyGate to persist fetched proxies
		if pg != nil {
			pg.SetPoolRepo(proxyListRepo)
			log.Println("manager: ProxyGate pool connected to database for persistence")

			// Load existing healthy proxies from database into memory pool
			ctx := context.Background()
			if err := pg.LoadFromDatabase(ctx); err != nil {
				log.Printf("manager: failed to load proxies from database: %v", err)
			}
		}
		log.Println("manager: ProxyListRepository initialized for proxy list access")
	}

	// Load sources if proxyRepo is available
	if proxyRepo != nil && pg != nil {
		ctx := context.Background()
		sources, err := proxyRepo.List(ctx)
		if err != nil {
			log.Printf("manager: failed to load proxy sources: %v", err)
		} else {
			count := 0
			for _, s := range sources {
				pg.AddSource(s.URL)
				count++
			}
			if count > 0 {
				log.Printf("manager: loaded %d proxy sources from database", count)
				// Trigger refresh to fetch from new sources
				go func() {
					if err := pg.Refresh(context.Background()); err != nil {
						log.Printf("manager: failed to refresh proxies after loading sources: %v", err)
					}
				}()
			}
		}
	}

	// Setup router
	router := api.NewRouter(jobHandler, workerHandler, statsHandler, proxyHandler, resultHandler, businessListingHandler)

	// Set cached handlers for read operations if available
	if cachedJobHandler != nil || cachedStatsHandler != nil || cachedResultHandler != nil {
		router.SetCachedHandlers(cachedJobHandler, cachedStatsHandler, cachedResultHandler)
	}
	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		apiToken = os.Getenv("API_KEY")
	}

	if apiToken == "" {
		log.Println("manager: WARNING - no API_TOKEN set, API will be unprotected!")
	} else {
		log.Println("manager: API_TOKEN configured")
	}

	handler := router.Setup(apiToken)

	var httpHandler http.Handler = handler

	// Serve static files if configured
	if cfg.StaticFolder != "" {
		log.Printf("manager: static folder configured at %s", cfg.StaticFolder)
		fs := http.FileServer(http.Dir(cfg.StaticFolder))
		httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Health check endpoint - always serve from API handler
			if r.URL.Path == "/health" || r.URL.Path == "/api/v2/health" {
				handler.ServeHTTP(w, r)
				return
			}

			// If API path, serve API
			if strings.HasPrefix(r.URL.Path, "/api") {
				handler.ServeHTTP(w, r)
				return
			}

			// If file exists, serve it
			path := filepath.Join(cfg.StaticFolder, r.URL.Path)
			// Check if it's a directory, if so, look for index.html inside?
			// FileServer handles directory index automatically if index.html exists.
			// But for SPA we want to fallback to root index.html for unknown routes.

			if _, err := os.Stat(path); err == nil {
				// File or directory exists
				// If directory and no index.html, it might list files (we disable listing usually)
				// But let's rely on FileServer.
				fs.ServeHTTP(w, r)
				return
			}

			// If not found and not API, serve index.html (SPA)
			http.ServeFile(w, r, filepath.Join(cfg.StaticFolder, "index.html"))
		})
	}

	// Create HTTP server
	// WriteTimeout set to 6 minutes to accommodate large downloads
	// (results exports can have 100k+ records)
	srv := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpHandler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      6 * time.Minute,
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
		proxyGate: pg,
		jobQueue:  jobQueue,
		mqPub:     mqPublisher,
		cache:     redisCache,
		spawner:   workerSpawner,
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
	if m.spawner != nil {
		m.spawner.Close()
	}
	if m.mqPub != nil {
		m.mqPub.Close()
	}
	if m.cache != nil {
		m.cache.Close()
	}
	if m.jobQueue != nil {
		m.jobQueue.Close()
	}
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
