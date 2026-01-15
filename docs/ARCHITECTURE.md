# Architecture Documentation

This document provides comprehensive technical details about the Google Maps Scraper architecture, including database normalization, caching layer, DSN bridge mechanism, worker deduplication, and API endpoints.

## Table of Contents

1. [Database Normalization Layer](#1-database-normalization-layer)
2. [Cache Implementation](#2-cache-implementation)
3. [DSN Bridge Mechanism](#3-dsn-bridge-mechanism)
4. [Worker Deduplication](#4-worker-deduplication)
5. [API Endpoints](#5-api-endpoints)
6. [Message Queue Architecture](#6-message-queue-architecture)

---

## 1. Database Normalization Layer

### Overview

The system uses a normalized database schema to efficiently store and query business listings. Raw scraping results are stored as JSONB in the `results` table, then automatically extracted into normalized tables via PostgreSQL triggers.

### Tables

#### `business_listings`
Primary table for normalized business data extracted from `results.data` JSONB.

```sql
CREATE TABLE business_listings (
    id BIGSERIAL PRIMARY KEY,
    result_id BIGINT NOT NULL REFERENCES results(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs_queue(id) ON DELETE SET NULL,
    place_id TEXT,
    cid TEXT,
    data_id TEXT,
    title TEXT NOT NULL,
    category TEXT,
    categories TEXT[],
    address TEXT,
    phone TEXT,
    website TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    plus_code TEXT,
    timezone TEXT,
    address_street TEXT,
    address_city TEXT,
    address_state TEXT,
    address_postal_code TEXT,
    address_country TEXT,
    review_count INTEGER DEFAULT 0,
    review_rating NUMERIC(3,1),
    status TEXT,
    price_range TEXT,
    description TEXT,
    link TEXT,
    reviews_link TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Key Indexes:**
- `idx_business_listings_title_trgm` - Trigram index for fuzzy text search
- `idx_business_listings_address_trgm` - Trigram index for address search
- `idx_business_listings_category` - B-tree index for category filtering
- `idx_business_listings_city`, `idx_business_listings_country` - Location filtering
- `idx_business_listings_review_rating` - Sorting by rating

#### `emails`
Deduplicated email storage with validation metadata from Moribouncer API.

```sql
CREATE TABLE emails (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    domain TEXT GENERATED ALWAYS AS (...) STORED,
    local_part TEXT GENERATED ALWAYS AS (...) STORED,
    validation_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (validation_status IN (
            'pending', 'local_valid', 'local_invalid',
            'api_valid', 'api_invalid', 'api_error', 'api_skipped'
        )),
    local_validation_passed BOOLEAN,
    local_validation_reason TEXT,
    local_validated_at TIMESTAMPTZ,
    api_status TEXT,
    api_score NUMERIC(5,2),
    api_deliverable BOOLEAN,
    api_disposable BOOLEAN,
    api_role_account BOOLEAN,
    api_free_email BOOLEAN,
    api_catch_all BOOLEAN,
    api_reason TEXT,
    api_validated_at TIMESTAMPTZ,
    is_acceptable BOOLEAN GENERATED ALWAYS AS (...) STORED,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    occurrence_count INTEGER DEFAULT 1
);
```

**Validation Status Flow:**
```
pending → local_valid/local_invalid → api_valid/api_invalid/api_error
```

**`is_acceptable` Computed Column Logic:**
- `api_valid` → `true`
- `api_invalid` → `false`
- `api_error` or `api_skipped` → falls back to `local_validation_passed`
- `local_valid` → `true`
- `local_invalid` → `false`

#### `business_emails`
Junction table linking businesses to emails (many-to-many relationship).

```sql
CREATE TABLE business_emails (
    id BIGSERIAL PRIMARY KEY,
    business_listing_id BIGINT NOT NULL REFERENCES business_listings(id),
    email_id BIGINT NOT NULL REFERENCES emails(id),
    source TEXT DEFAULT 'website',  -- 'website' or 'google_maps'
    position INTEGER DEFAULT 0,      -- Order in original list
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(business_listing_id, email_id)
);
```

### Trigger-Based Auto-Population

When a new result is inserted into the `results` table, the `trg_populate_normalized_listings` trigger automatically:

1. Extracts business data from `results.data` JSONB
2. Inserts/updates into `business_listings`
3. Parses email array and validation metadata from `email_validations`
4. Upserts emails into `emails` table with API validation results
5. Creates junction records in `business_emails`

```sql
CREATE TRIGGER trg_populate_normalized_listings
    AFTER INSERT ON results
    FOR EACH ROW
    EXECUTE FUNCTION populate_normalized_listings();
```

### Email Validation Pipeline

The `gmaps.Entry` struct now includes `EmailValidations` field:

```go
// gmaps/entry.go
type Entry struct {
    // ... other fields
    Emails           []string          `json:"emails,omitempty"`
    EmailValidations []EmailValidation `json:"email_validations,omitempty"`
}

type EmailValidation struct {
    Email       string  `json:"email"`
    Status      string  `json:"status"`       // "valid", "invalid", "api_error"
    Score       float64 `json:"score"`        // 0-100
    Deliverable bool    `json:"deliverable"`
    Disposable  bool    `json:"disposable"`
    RoleAccount bool    `json:"role_account"`
    FreeEmail   bool    `json:"free_email"`
    CatchAll    bool    `json:"catch_all"`
    Reason      string  `json:"reason"`
}
```

### Views for Analytics

#### `v_business_listings_with_emails`
Pre-aggregated view joining listings with their emails:

```sql
SELECT bl.*,
    emails_with_validation AS jsonb,
    emails AS text[],
    valid_email_count,
    total_email_count
FROM business_listings bl
LEFT JOIN business_emails be ON ...
LEFT JOIN emails e ON ...
GROUP BY bl.id;
```

#### `v_email_validation_queue`
Priority queue for pending email validations:

```sql
SELECT e.id, e.email, e.domain, e.validation_status,
    ROW_NUMBER() OVER (ORDER BY occurrence_count DESC, first_seen_at ASC) as priority
FROM emails e
WHERE validation_status IN ('pending', 'local_valid');
```

### Backfill Function

For migrating existing data:

```sql
SELECT * FROM backfill_normalized_listings(5000);  -- Process 5000 at a time
-- Returns: (processed INTEGER, errors INTEGER)
```

---

## 2. Cache Implementation

### Architecture

The caching layer uses Redis to reduce database load for dashboard read operations.

**Location:** `internal/cache/`

### Interface

```go
// internal/cache/cache.go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeleteByPattern(ctx context.Context, pattern string) error
    Close() error
}
```

### Implementations

| Implementation | File | Description |
|----------------|------|-------------|
| `RedisCache` | `redis.go` | Production Redis client |
| `NoOpCache` | `noop.go` | No-op fallback when Redis unavailable |

### Key Prefixes

```go
const (
    KeyPrefixDashboardStats   = "cache:dashboard:stats"
    KeyPrefixDashboardJobs    = "cache:dashboard:jobs"
    KeyPrefixDashboardResults = "cache:dashboard:results"
    KeyPrefixDashboardSearch  = "cache:dashboard:search"
)
```

### TTL Configuration

```go
const (
    TTLStats     = 30 * time.Second   // Dashboard statistics
    TTLJobsList  = 60 * time.Second   // Job listings
    TTLJobDetail = 120 * time.Second  // Single job details
    TTLResults   = 60 * time.Second   // Result listings
    TTLSearch    = 30 * time.Second   // Search results
)
```

### Cache Key Patterns

| Endpoint | Cache Key Pattern |
|----------|-------------------|
| `GET /api/v2/stats` | `cache:dashboard:stats` |
| `GET /api/v2/jobs` | `cache:dashboard:jobs:list:page={N}:perPage={N}:status={S}` |
| `GET /api/v2/jobs/{id}` | `cache:dashboard:jobs:{uuid}` |
| `GET /api/v2/jobs/{id}/results` | `cache:dashboard:results:{uuid}:page={N}:perPage={N}` |
| `GET /api/v2/results` | `cache:dashboard:results:all:page={N}:perPage={N}` |
| `GET /api/v2/jobs/stats` | `cache:dashboard:jobs:stats` |

### Cached vs Standard Handlers

The router uses cached handlers for read operations when Redis is available:

```go
// internal/api/router.go
type Router struct {
    // Standard handlers (always present)
    jobs    *handlers.JobHandler
    workers *handlers.WorkerHandler
    stats   *handlers.StatsHandler

    // Cached handlers (optional, for read operations)
    cachedJobs    *handlers.CachedJobHandler
    cachedStats   *handlers.CachedStatsHandler
    cachedResults *handlers.CachedResultHandler
}
```

**Selection Logic:**
```go
// router.go:handleJobs
switch req.Method {
case http.MethodGet:
    if r.cachedJobs != nil {
        r.cachedJobs.List(w, req)  // Use cached
    } else {
        r.jobs.List(w, req)        // Use standard
    }
case http.MethodPost:
    r.jobs.Create(w, req)          // Always standard (writes)
}
```

### Cache Invalidation Strategy

**Location:** `internal/api/handlers/cached.go`

The `CacheInvalidator` component handles cache invalidation:

```go
type CacheInvalidator struct {
    cache cache.Cache
}

// Called when new results are added
func (ci *CacheInvalidator) InvalidateOnNewResults(ctx context.Context, jobID uuid.UUID) error {
    // Invalidate job results
    ci.cache.DeleteByPattern(ctx, fmt.Sprintf("%s:%s:*", KeyPrefixDashboardResults, jobID))
    // Invalidate global results
    ci.cache.DeleteByPattern(ctx, KeyPrefixDashboardResults+":all:*")
    // Invalidate search cache
    ci.cache.DeleteByPattern(ctx, KeyPrefixDashboardSearch+":*")
    // Invalidate stats
    ci.cache.Delete(ctx, KeyPrefixDashboardStats)
    return nil
}

// Called when job status changes
func (ci *CacheInvalidator) InvalidateOnJobStatusChange(ctx context.Context, jobID uuid.UUID) error {
    ci.cache.Delete(ctx, fmt.Sprintf("%s:%s", KeyPrefixDashboardJobs, jobID))
    ci.cache.DeleteByPattern(ctx, KeyPrefixDashboardJobs+":list:*")
    ci.cache.Delete(ctx, KeyPrefixDashboardJobs+":stats")
    return nil
}
```

---

## 3. DSN Bridge Mechanism

### Overview

The DSN Bridge connects the Dashboard (`jobs_queue` table) to the legacy DSN workers (`gmaps_jobs` table). This enables Dashboard-created jobs to be processed by existing DSN-mode workers.

### Components

#### `GmapsJobPusher` Interface

**Location:** `postgres/provider.go`

```go
type GmapsJobPusher interface {
    Push(ctx context.Context, job scrapemate.IJob) error
    PushWithParent(ctx context.Context, job scrapemate.IJob, parentID string) error
}
```

#### `PushWithParent` Function

Inserts jobs into `gmaps_jobs` with a reference back to the Dashboard job:

```go
func (p *provider) PushWithParent(ctx context.Context, job scrapemate.IJob, parentID string) error {
    q := `INSERT INTO gmaps_jobs
        (id, priority, payload_type, payload, created_at, status, parent_job_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`

    // Serialize job using gob encoding
    // parentID links back to jobs_queue.id
    _, err := p.db.ExecContext(ctx, q,
        job.GetID(), job.GetPriority(), payloadType,
        buf.Bytes(), time.Now().UTC(), "new", parentIDArg)
    return err
}
```

#### `bridgeToGmapsJobs` in JobService

**Location:** `internal/service/job.go`

```go
func (s *JobService) bridgeToGmapsJobs(ctx context.Context, jobID uuid.UUID, queries []string, opts domain.JobOptions) error {
    if s.gmapsPusher == nil {
        return nil // No bridge configured
    }

    for _, query := range queries {
        // Create GmapJob with ParentID
        gmapJob := gmaps.NewGmapJob(langCode, query, ...)
        gmapJob.ParentID = jobID.String()

        // Push to gmaps_jobs table
        if err := s.gmapsPusher.PushWithParent(ctx, gmapJob, jobID.String()); err != nil {
            return err
        }
    }
    return nil
}
```

### Auto-Migration System

**Location:** `internal/migration/`

The auto-migration system detects database state and applies appropriate migrations:

```go
// internal/migration/automigrate.go
type MigrationState int

const (
    StateFreshInstall      MigrationState = iota  // No tables exist
    StateBothExistUnlinked                        // Both tables, no parent_job_id
    StateOnlyGmapsJobs                            // Only gmaps_jobs exists
    StateOnlyJobsQueue                            // Only jobs_queue exists
    StateAlreadyMigrated                          // parent_job_id column exists
)

func AutoMigrate(ctx context.Context, db *sql.DB) error {
    state, _ := DetectMigrationState(ctx, db)

    switch state {
    case StateAlreadyMigrated:
        return nil  // Nothing to do
    case StateBothExistUnlinked:
        return migrateBothExistUnlinked(ctx, db)  // Add parent_job_id column
    case StateOnlyGmapsJobs:
        return migrateOnlyGmapsJobs(ctx, db)      // Add parent_job_id column
    case StateOnlyJobsQueue:
        return migrateOnlyJobsQueue(ctx, db)      // Create gmaps_jobs + bridge
    }
    return nil
}
```

### Backward Compatibility

**DSN workers (`-dsn` mode) continue to work unchanged:**

1. Workers query `gmaps_jobs` table for jobs with `status = 'new'`
2. The `parent_job_id` column is optional (nullable)
3. Jobs without `parent_job_id` are CLI-originated
4. Jobs with `parent_job_id` are Dashboard-originated

**Results flow:**
```
DSN Worker → results table (with parent_job_id in data)
           → trigger → business_listings + emails
           → Dashboard shows results
```

---

## 4. Worker Deduplication

### Overview

The distributed deduplicator prevents processing the same Google Maps place multiple times across workers.

**Location:** `internal/queue/deduper.go`

### Interface

```go
type Deduper struct {
    client *redis.Client
    prefix string        // Default: "dedup"
    ttl    time.Duration // Default: 24 hours
}

// Check if place has been scraped (returns true if duplicate)
func (d *Deduper) IsDuplicate(ctx context.Context, placeID string) (bool, error)

// Check if URL has been processed
func (d *Deduper) IsDuplicateURL(ctx context.Context, url string) (bool, error)

// Mark place as seen
func (d *Deduper) MarkAsSeen(ctx context.Context, placeID string) error

// Compatible with scrapemate deduper interface
func (d *Deduper) Seen(id string) bool
func (d *Deduper) AddIfNotExists(ctx context.Context, key string) bool
```

### Redis Key Patterns

| Type | Key Pattern | TTL |
|------|-------------|-----|
| Place ID | `dedup:place:{placeID}` | 24 hours |
| URL | `dedup:url:{url}` | 24 hours |

### Algorithm

Uses Redis `SETNX` (Set if Not eXists) for atomic check-and-set:

```go
func (d *Deduper) IsDuplicate(ctx context.Context, placeID string) (bool, error) {
    key := fmt.Sprintf("%s:place:%s", d.prefix, placeID)

    // SETNX returns true if key was set (new), false if existed (duplicate)
    wasSet, err := d.client.SetNX(ctx, key, 1, d.ttl).Result()
    if err != nil {
        return false, err
    }

    // If wasSet is true → NOT a duplicate (first time seeing)
    // If wasSet is false → IS a duplicate (already existed)
    return !wasSet, nil
}
```

### Configuration

```go
type DedupeConfig struct {
    RedisURL  string        // e.g., "redis://localhost:6379"
    RedisAddr string        // e.g., "localhost:6379"
    Password  string
    DB        int
    Prefix    string        // Default: "dedup"
    TTL       time.Duration // Default: 24 hours
}
```

### Connection Pool Settings

```go
redis.Options{
    PoolSize:     10,
    MinIdleConns: 2,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,
}
```

---

## 5. API Endpoints

### Jobs API

| Method | Endpoint | Description | Cached |
|--------|----------|-------------|--------|
| GET | `/api/v2/jobs` | List jobs with pagination | ✓ |
| POST | `/api/v2/jobs` | Create new job | ✗ |
| GET | `/api/v2/jobs/stats` | Job statistics | ✓ |
| GET | `/api/v2/jobs/{id}` | Get job details | ✓ |
| DELETE | `/api/v2/jobs/{id}` | Delete job | ✗ |
| POST | `/api/v2/jobs/{id}/pause` | Pause job | ✗ |
| POST | `/api/v2/jobs/{id}/resume` | Resume job | ✗ |
| POST | `/api/v2/jobs/{id}/cancel` | Cancel job | ✗ |
| GET | `/api/v2/jobs/{id}/results` | Get job results | ✓ |
| POST | `/api/v2/jobs/{id}/results` | Submit results (from workers) | ✗ |
| GET | `/api/v2/jobs/{id}/download` | Download results as CSV/JSON/XLSX | ✗ |

#### POST `/api/v2/jobs/{id}/results` (Result Submission)

Workers submit scraped results to this endpoint:

```json
POST /api/v2/jobs/{job-uuid}/results
Content-Type: application/json

{
    "worker_id": "worker-123",
    "results": [
        {
            "title": "Business Name",
            "address": "123 Main St",
            "phone": "+1234567890",
            "website": "https://example.com",
            "emails": ["info@example.com"],
            "email_validations": [
                {
                    "email": "info@example.com",
                    "status": "valid",
                    "score": 85.5,
                    "deliverable": true,
                    "disposable": false,
                    "role_account": false
                }
            ],
            "review_count": 42,
            "review_rating": 4.5,
            "latitude": 37.7749,
            "longitude": -122.4194
        }
    ]
}
```

### Workers API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v2/workers` | List all workers |
| POST | `/api/v2/workers/register` | Register new worker |
| POST | `/api/v2/workers/heartbeat` | Update worker heartbeat |
| GET | `/api/v2/workers/stats` | Worker statistics |
| GET | `/api/v2/workers/{id}` | Get worker details |
| DELETE | `/api/v2/workers/{id}` | Unregister worker |
| POST | `/api/v2/workers/{id}/claim` | Claim a job |
| POST | `/api/v2/workers/{id}/complete` | Mark job complete |
| POST | `/api/v2/workers/{id}/fail` | Mark job failed |
| POST | `/api/v2/workers/{id}/release` | Release claimed job |

#### Worker Registration Flow

```
1. Worker starts up
   POST /api/v2/workers/register
   Body: {"worker_id": "worker-hostname-uuid"}
   Response: 201 Created + Worker object

2. Worker sends periodic heartbeats (every 10s)
   POST /api/v2/workers/heartbeat
   Body: {
       "worker_id": "worker-hostname-uuid",
       "hostname": "worker-host",
       "status": "idle",  // or "busy"
       "current_job_id": null  // or UUID if processing
   }
   Response: 204 No Content

3. Worker claims a job
   POST /api/v2/workers/{worker_id}/claim
   Response: {"job": {...}} or {"job": null}

4. Worker completes job
   POST /api/v2/workers/{worker_id}/complete
   Body: {"job_id": "uuid", "places_scraped": 150}
   Response: 204 No Content

5. Worker gracefully shuts down
   DELETE /api/v2/workers/{worker_id}
   Response: 204 No Content
```

#### Heartbeat Mechanism

**Configuration (`internal/domain/worker.go`):**
```go
const HeartbeatTimeout = 30 * time.Second   // Worker offline after 30s no heartbeat
const HeartbeatInterval = 10 * time.Second  // Workers send heartbeat every 10s
```

**Monitor (`internal/heartbeat/monitor.go`):**

The Manager runs a background goroutine that:
1. Runs every `HeartbeatInterval` (10s)
2. Calls `WorkerService.MarkOfflineWorkers()`
3. Marks workers as `offline` if `last_heartbeat` > `HeartbeatTimeout`
4. Logs count of workers marked offline

```go
func (m *Monitor) Run(ctx context.Context) error {
    ticker := time.NewTicker(m.interval)
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            count, _ := m.workers.MarkOfflineWorkers(ctx)
            if count > 0 {
                log.Printf("marked %d workers as offline", count)
            }
        }
    }
}
```

### Results API

| Method | Endpoint | Description | Cached |
|--------|----------|-------------|--------|
| GET | `/api/v2/results` | List all results globally | ✓ |
| GET | `/api/v2/results/download` | Download all results | ✗ |

### Stats API

| Method | Endpoint | Description | Cached |
|--------|----------|-------------|--------|
| GET | `/api/v2/stats` | Dashboard statistics | ✓ |

### Health API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (no auth) |
| GET | `/api/v2/health` | Health check (no auth) |

---

## 6. Message Queue Architecture

### RabbitMQ Integration

**Location:** `internal/mq/`

#### Publisher (`publisher.go`)

```go
type Publisher interface {
    Publish(ctx context.Context, msg *JobMessage) error
    Close() error
}

type JobMessage struct {
    JobID    string `json:"job_id"`
    Priority int    `json:"priority"`
    Type     string `json:"type"`  // "job:process"
}
```

#### Consumer (`consumer.go`)

```go
type Consumer interface {
    Consume(ctx context.Context, handler func(context.Context, *JobMessage) error) error
    Close() error
}
```

### Queue Configuration

```
Exchange: (default)
Queues (by priority):
├── gmaps.jobs.critical  (priority >= 10)
├── gmaps.jobs.high      (priority >= 5)
├── gmaps.jobs.default   (priority 0-4)
└── gmaps.jobs.low       (priority < 0)
```

### Retry Mechanism

**Exponential backoff with max 5 retries:**

```go
const (
    initialBackoff = 1 * time.Second
    maxBackoff     = 30 * time.Second
    maxRetries     = 5
)

// Backoff calculation: 1s, 2s, 4s, 8s, 16s (capped at 30s)
backoff := initialBackoff * time.Duration(1<<uint(retryCount))
if backoff > maxBackoff {
    backoff = maxBackoff
}
```

**Retry flow:**
1. Message processing fails
2. Check retry count from `x-retry-count` header
3. If retries < maxRetries: wait backoff, republish with incremented count
4. If retries >= maxRetries: reject message (dead-letter)

### Docker Compose Configuration

```yaml
rabbitmq:
  image: rabbitmq:3-management-alpine
  container_name: gmaps-rabbitmq
  environment:
    RABBITMQ_DEFAULT_USER: gmaps
    RABBITMQ_DEFAULT_PASS: gmaps_secret
  ports:
    - "15672:15672"  # Management UI
  volumes:
    - rabbitmq_data:/var/lib/rabbitmq
  healthcheck:
    test: rabbitmq-diagnostics -q ping
  networks:
    - gmaps-network

manager:
  command:
    - "-rabbitmq-url=amqp://gmaps:gmaps_secret@rabbitmq:5672/"
    - "-redis-addr=redis:6379"

worker:
  command:
    - "-rabbitmq-url=amqp://gmaps:gmaps_secret@rabbitmq:5672/"
    - "-redis-addr=redis:6379"
```

---

## File Reference

| Component | Location |
|-----------|----------|
| Database normalization migration | `runner/managerrunner/migrations/0004_normalized_business_listings.up.sql` |
| Business listing repository | `internal/repository/postgres/business_listing.go` |
| Cache interface | `internal/cache/cache.go` |
| Redis cache implementation | `internal/cache/redis.go` |
| No-op cache fallback | `internal/cache/noop.go` |
| Cached handlers | `internal/api/handlers/cached.go` |
| DSN bridge (GmapsJobPusher) | `postgres/provider.go` |
| Auto-migration | `internal/migration/automigrate.go`, `executor.go` |
| Worker deduplication | `internal/queue/deduper.go` |
| Worker handlers | `internal/api/handlers/workers.go` |
| Heartbeat monitor | `internal/heartbeat/monitor.go` |
| RabbitMQ publisher | `internal/mq/publisher.go` |
| RabbitMQ consumer | `internal/mq/consumer.go` |
| API router | `internal/api/router.go` |
| Domain models | `internal/domain/` |
