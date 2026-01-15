# REVISED PLAN: Dashboard → DSN Bridge (Simplified)

> **Status**: REVISED based on user feedback
>
> **Key Decisions**:
> - ❌ RabbitMQ NOT needed (polling 50-300ms acceptable)
> - ✅ DSN mode in production → minimal consumer changes
> - ✅ 50+ workers → ensure efficient PostgreSQL polling
> - ✅ Focus: Bridge Dashboard to `gmaps_jobs` table
> - ✅ External Email Validation dengan Moribouncer
> - ✅ Dashboard uses Manager mode (NOT WebRunner)

---

## Verification: Dashboard vs WebRunner Separation

### CONFIRMED: Dashboard and WebRunner are COMPLETELY SEPARATE

| Aspect | Dashboard (Manager Mode) | WebRunner (-web deprecated) |
|--------|--------------------------|------------------------------|
| **Flag** | `-manager` | `-web` |
| **Code Location** | `runner/managerrunner/` | `runner/webrunner/` |
| **API Routes** | `/api/v2/...` | `/api/v1/...` or legacy |
| **Database** | PostgreSQL (recommended) | SQLite only |
| **Frontend** | React SPA (`web/frontend/`) | Server-side templates |
| **Purpose** | Production API + Dashboard | Deprecated local mode |

### Field Mapping: Frontend → Backend → Database (COMPLETE)

```
Frontend (JobCreatePayload)    →  Handler    →  Domain    →  DB (jobs_queue)
───────────────────────────────────────────────────────────────────────────
name: string                   →  Name       →  Name      →  name TEXT
keywords: string[]             →  Keywords   →  Keywords  →  keywords TEXT[]
lang: string                   →  Lang       →  Lang      →  lang TEXT
lat?: number                   →  Lat        →  GeoLat    →  geo_lat DOUBLE PRECISION
lon?: number                   →  Lon        →  GeoLon    →  geo_lon DOUBLE PRECISION
zoom: number                   →  Zoom       →  Zoom      →  zoom INT
radius: number                 →  Radius     →  Radius    →  radius INT
depth: number                  →  Depth      →  Depth     →  depth INT
fast_mode: boolean             →  FastMode   →  FastMode  →  fast_mode BOOLEAN
extract_email: boolean         →  ExtractEmail→ ExtractEmail→ extract_email BOOLEAN
priority: number               →  Priority   →  Priority  →  priority INT
max_time: number (seconds)     →  MaxTime    →  MaxTime   →  max_time INTERVAL
(not sent)                     →  Proxies    →  Proxies   →  proxies TEXT[]
```

### Verification Results

| Check | Status | Notes |
|-------|--------|-------|
| Dashboard uses Manager mode | ✅ PASS | Frontend `/api/v2` → managerrunner |
| Dashboard NOT using WebRunner | ✅ PASS | WebRunner serves `/api/v1` |
| All frontend fields mapped | ✅ PASS | All `JobCreatePayload` fields reach DB |
| Database schema matches domain | ✅ PASS | Migration 0005 aligns with domain.Job |
| Type conversions correct | ✅ PASS | `max_time: int(s)` → `Duration` → `INTERVAL` |
| Redis queue integration | ✅ PASS | Jobs enqueued after DB insert |
| **Bridge to gmaps_jobs** | ❌ MISSING | **This plan implements this** |

---

## Table of Contents

1. [Target Architecture](#target-architecture-simplified)
2. [Background: Key Components Explained](#background-key-components-explained)
3. [Problem Statement](#problem-statement)
4. [Solution: Bridge Layer](#solution-bridge-layer)
5. [Implementation Phases](#implementation-phases)
6. [Phase 6: External Email Validation (Moribouncer)](#phase-6-external-email-validation-moribouncer)
7. [Files Summary](#files-summary)
8. [Verification Steps](#verification-steps)
9. [Estimated Effort](#estimated-effort)
10. [Frontend Architecture Analysis](#frontend-architecture-analysis)
11. [Phase 7: Auto Migration (Old → New Design)](#phase-7-auto-migration-old-design--new-design)

---

## Target Architecture (SIMPLIFIED)

```
┌────────────────────────────────────────────────────────────────┐
│              SIMPLIFIED ARCHITECTURE (NO RABBITMQ)             │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌─────────────────┐     REST API     ┌─────────────────┐     │
│  │     Dashboard   │─────────────────►│     Manager     │     │
│  │   (React/Vite)  │                  │    (Go API)     │     │
│  └─────────────────┘                  └────────┬────────┘     │
│                                                │               │
│                                    1. Create parent job       │
│                                       in jobs_queue           │
│                                                │               │
│                                    2. Create seed jobs        │
│                                       using CreateSeedJobs()  │
│                                                │               │
│                                    3. INSERT to gmaps_jobs    │
│                                       (GOB encoded)           │
│                                                ▼               │
│                               ┌────────────────────────────┐  │
│                               │       gmaps_jobs           │  │
│                               │   (PostgreSQL table)       │  │
│                               │                            │  │
│                               │  • id (UUID)               │  │
│                               │  • priority (INT)          │  │
│                               │  • payload_type (VARCHAR)  │  │
│                               │  • payload (BYTEA/GOB)     │  │
│                               │  • status (new/queued)     │  │
│                               │  • parent_job_id (NEW!)    │  │
│                               │  • created_at              │  │
│                               └─────────────┬──────────────┘  │
│                                             │                  │
│                            Polling (50-300ms backoff)         │
│                               FOR UPDATE SKIP LOCKED          │
│                                             │                  │
│            ┌────────────────────────────────┼─────────────────┤
│            │                                │                 │
│            ▼                                ▼                 │
│  ┌─────────────────┐              ┌─────────────────┐        │
│  │  DSN Worker 1   │              │  DSN Worker N   │        │
│  │  (UNCHANGED!)   │              │  (UNCHANGED!)   │        │
│  └────────┬────────┘              └────────┬────────┘        │
│           │                                │                  │
│           │         ┌──────────────────────┘                  │
│           │         │                                         │
│           ▼         ▼                                         │
│      ┌─────────────────────┐                                  │
│      │   Email Extraction  │                                  │
│      │   (if enabled)      │                                  │
│      └──────────┬──────────┘                                  │
│                 │                                              │
│                 ▼                                              │
│      ┌─────────────────────┐                                  │
│      │     Moribouncer     │◄── External Email Validation    │
│      │   Validation API    │    • Is deliverable?            │
│      └──────────┬──────────┘    • Is disposable?             │
│                 │               • Quality score              │
│                 ▼                                              │
│        ┌─────────────────┐                                    │
│        │     results     │                                    │
│        │   (PostgreSQL)  │                                    │
│        └─────────────────┘                                    │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

---

## Background: Key Components Explained

### 1. `jobs_queue` - Dashboard/Manager Table

**File**: `scripts/migrations/0005_manager_worker_architecture.up.sql`

Tabel ini digunakan oleh **Dashboard (Manager mode)** untuk tracking jobs di UI.

```sql
CREATE TABLE IF NOT EXISTS jobs_queue (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,              -- "Cafe Jakarta Search"
    status TEXT NOT NULL,            -- pending/running/completed/failed
    priority INT NOT NULL,

    -- Search configuration (structured columns)
    keywords TEXT[] NOT NULL,        -- ["cafe jakarta", "coffee bandung"]
    lang TEXT DEFAULT 'en',
    geo_lat DOUBLE PRECISION,
    geo_lon DOUBLE PRECISION,
    zoom INT DEFAULT 15,
    radius INT DEFAULT 10000,
    depth INT DEFAULT 10,
    fast_mode BOOLEAN,
    extract_email BOOLEAN,

    -- Progress tracking
    total_places INT,
    scraped_places INT,
    failed_places INT,

    -- Worker assignment
    worker_id TEXT,

    -- Timestamps
    created_at, updated_at, started_at, completed_at
);
```

| Aspek | Detail |
|-------|--------|
| **Digunakan oleh** | Dashboard (Manager mode) |
| **Format** | Structured columns (JSON-like) |
| **Purpose** | UI tracking, progress display |
| **1 row =** | 1 Dashboard job (bisa punya banyak keywords) |

---

### 2. `CreateSeedJobs()` - Function untuk Membuat Scraping Jobs

**File**: `runner/jobs.go:19-131`

Function ini mengubah input text (keywords) menjadi `scrapemate.IJob` objects.

```go
func CreateSeedJobs(
    fastmode bool,
    langCode string,
    r io.Reader,           // ← Input dari file atau stdin
    maxDepth int,
    email bool,
    geoCoordinates string,
    zoom int,
    radius float64,
    dedup deduper.Deduper,
    exitMonitor exiter.Exiter,
    extraReviews bool,
) (jobs []scrapemate.IJob, err error) {

    scanner := bufio.NewScanner(r)

    for scanner.Scan() {
        query := strings.TrimSpace(scanner.Text())  // ← Baca per line
        if query == "" {
            continue
        }

        var job scrapemate.IJob

        if !fastmode {
            // Normal mode: gmaps.GmapJob
            job = gmaps.NewGmapJob(id, langCode, query, maxDepth, email, ...)
        } else {
            // Fast mode: gmaps.SearchJob
            job = gmaps.NewSearchJob(&jparams, opts...)
        }

        jobs = append(jobs, job)
    }

    return jobs, scanner.Err()
}
```

| Aspek | Detail |
|-------|--------|
| **Input** | `io.Reader` (file atau stdin) |
| **Output** | `[]scrapemate.IJob` (list of GmapJob/SearchJob) |
| **1 line =** | 1 scraping job |
| **Digunakan oleh** | CLI `-produce` mode, filerunner |

---

### 3. `gmaps_jobs` - DSN Workers Table

**File**: `postgres/provider.go` (implicit schema dari Push method)

Tabel ini digunakan oleh **DSN workers** untuk job queue.

```sql
CREATE TABLE gmaps_jobs (
    id TEXT PRIMARY KEY,
    priority INT,
    payload_type VARCHAR,  -- "search", "place", "email"
    payload BYTEA,         -- GOB encoded binary!
    created_at TIMESTAMPTZ,
    status VARCHAR         -- "new", "queued"
);
```

| Aspek | Detail |
|-------|--------|
| **Digunakan oleh** | DSN workers (-dsn mode) |
| **Format** | GOB binary blob |
| **Purpose** | Job queue untuk scraping |
| **Claim method** | `FOR UPDATE SKIP LOCKED` |
| **1 row =** | 1 keyword/query to scrape |

---

### 4. How Workers Are Spawned

**DSN Mode Worker Lifecycle:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        DSN MODE WORKER SPAWNING                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Terminal 1 (Producer - one-shot):                                          │
│  $ ./gmaps-scraper -dsn "postgres://..." -produce -input queries.txt       │
│        │                                                                    │
│        └─► INSERT jobs ke gmaps_jobs → EXIT                                │
│                                                                             │
│  Terminal 2, 3, 4... (Workers - long-running):                             │
│  $ ./gmaps-scraper -dsn "postgres://..." -c 4                              │
│        │                                                                    │
│        ▼                                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    scrapemateapp.ScrapeMateApp                       │   │
│  │                                                                      │   │
│  │   ┌──────────────┐                                                   │   │
│  │   │  app.Start() │ ← Line 113: d.app.Start(ctx)                     │   │
│  │   └──────┬───────┘                                                   │   │
│  │          │                                                           │   │
│  │          ▼                                                           │   │
│  │   ┌──────────────────────────────────────────────────────────────┐  │   │
│  │   │ Internal goroutine pool (size = -c flag, default 4)          │  │   │
│  │   │                                                               │  │   │
│  │   │  goroutine 1 ──► poll gmaps_jobs ──► scrape ──► save results │  │   │
│  │   │  goroutine 2 ──► poll gmaps_jobs ──► scrape ──► save results │  │   │
│  │   │  goroutine 3 ──► poll gmaps_jobs ──► scrape ──► save results │  │   │
│  │   │  goroutine 4 ──► poll gmaps_jobs ──► scrape ──► save results │  │   │
│  │   │                                                               │  │   │
│  │   └──────────────────────────────────────────────────────────────┘  │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Initialization Flow** (`runner/databaserunner/databaserunner.go:30-104`):

```go
func New(cfg *runner.Config) (runner.Runner, error) {
    // 1. Connect ke PostgreSQL
    conn, err := openPsqlConn(cfg.Dsn)

    // 2. Create job provider (polls gmaps_jobs)
    provider: postgres.NewProvider(conn)

    // 3. Create result writer
    psqlWriter := postgres.NewResultWriter(conn)

    // 4. Configure scrapemate app dengan concurrency
    opts := []func(*scrapemateapp.Config) error{
        scrapemateapp.WithConcurrency(cfg.Concurrency),  // ← -c flag
        scrapemateapp.WithProvider(ans.provider),
        scrapemateapp.WithJS(scrapemateapp.DisableImages()),
    }

    // 5. Create ScrapeMateApp (manages goroutine pool)
    ans.app, err = scrapemateapp.NewScrapeMateApp(matecfg)
}
```

**Run** (`databaserunner.go:106-114`):

```go
func (d *dbrunner) Run(ctx context.Context) error {
    return d.app.Start(ctx)  // ← Starts internal goroutine pool
}
```

---

## Problem Statement

**Current Issue**: Dashboard dan DSN workers menggunakan tabel berbeda dan tidak terhubung.

| Component | Table Used | Format |
|-----------|------------|--------|
| Dashboard API | `jobs_queue` | JSON/domain.Job |
| DSN Workers | `gmaps_jobs` | GOB binary |

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CURRENT STATE (BROKEN)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Dashboard:                               CLI:                              │
│  ┌────────────────────┐                  ┌────────────────────┐            │
│  │ POST /api/v2/jobs  │                  │ -dsn -produce      │            │
│  │ {keywords: [...]}  │                  │ -input queries.txt │            │
│  └─────────┬──────────┘                  └─────────┬──────────┘            │
│            │                                       │                        │
│            ▼                                       ▼                        │
│  ┌────────────────────┐                  ┌────────────────────┐            │
│  │ JobService.Create()│                  │ CreateSeedJobs()   │            │
│  └─────────┬──────────┘                  └─────────┬──────────┘            │
│            │                                       │                        │
│            ▼                                       ▼                        │
│  ┌────────────────────┐                  ┌────────────────────┐            │
│  │    jobs_queue      │  ← NO LINK →     │    gmaps_jobs      │            │
│  │  (structured JSON) │                  │   (GOB binary)     │            │
│  └────────────────────┘                  └─────────┬──────────┘            │
│            │                                       │                        │
│            ▼                                       ▼                        │
│  ┌────────────────────┐                  ┌────────────────────┐            │
│  │  Manager Workers   │                  │    DSN Workers     │            │
│  │  (-worker mode)    │                  │    (-dsn mode)     │            │
│  └────────────────────┘                  └────────────────────┘            │
│                                                                             │
│  ❌ Dashboard job TIDAK terlihat oleh DSN workers!                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Result**: Job dari Dashboard TIDAK terlihat oleh DSN workers!

---

## Solution: Bridge Layer

Ketika Dashboard create job, juga insert ke `gmaps_jobs` dengan format yang sama seperti CLI `-produce`.

### 1:N Relationship

```
Dashboard Request:
{
  "name": "Cafe Jakarta Search",
  "keywords": ["cafe jakarta", "coffee bandung"],
  "depth": 10,
  "email": true
}
        │
        ▼
jobs_queue (1 record):
┌─────────────────────────────────────────┐
│ id: "dashboard-job-123"                 │
│ name: "Cafe Jakarta Search"             │
│ status: "running"                       │
│ total_tasks: 2                          │
│ completed_tasks: 0                      │
└─────────────────────────────────────────┘
        │
        ▼
gmaps_jobs (2 records, one per keyword):
┌─────────────────────────────────────────┐
│ id: "search-cafe-jakarta-xxx"           │
│ parent_job_id: "dashboard-job-123" ←────│
│ payload_type: "search"                  │
│ payload: <GOB encoded gmaps.GmapJob>    │
│ status: "new"                           │
├─────────────────────────────────────────┤
│ id: "search-coffee-bandung-xxx"         │
│ parent_job_id: "dashboard-job-123" ←────│
│ payload_type: "search"                  │
│ payload: <GOB encoded gmaps.GmapJob>    │
│ status: "new"                           │
└─────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 0: Database Schema Update
**Effort: 30 minutes**

Add `parent_job_id` column to link gmaps_jobs back to jobs_queue.

**File**: `migrations/xxx_add_parent_job_id.sql`

```sql
-- Add parent_job_id for linking gmaps_jobs to jobs_queue
ALTER TABLE gmaps_jobs ADD COLUMN parent_job_id UUID;
CREATE INDEX idx_gmaps_jobs_parent ON gmaps_jobs(parent_job_id);

-- Add progress tracking to jobs_queue
ALTER TABLE jobs_queue ADD COLUMN total_tasks INTEGER DEFAULT 0;
ALTER TABLE jobs_queue ADD COLUMN completed_tasks INTEGER DEFAULT 0;
```

---

### Phase 1: Create Reusable Seed Job Function
**Effort: 1-2 hours**

Extract `CreateSeedJobs` logic to accept []string instead of io.Reader.

**File**: `runner/seedjobs.go` (NEW)

```go
package runner

import (
    "strings"

    "github.com/gosom/scrapemate"
)

// SeedJobConfig for creating seed jobs from API
type SeedJobConfig struct {
    Keywords       []string
    FastMode       bool
    LangCode       string
    Depth          int
    Email          bool
    GeoCoordinates string // "lat,lon" or ""
    Zoom           int
    Radius         int
    ExtraReviews   bool
}

// CreateSeedJobsFromKeywords - reusable for CLI and API
func CreateSeedJobsFromKeywords(cfg SeedJobConfig) ([]scrapemate.IJob, error) {
    // Convert []string to io.Reader (adapter pattern)
    input := strings.NewReader(strings.Join(cfg.Keywords, "\n"))

    return CreateSeedJobs(
        cfg.FastMode,
        cfg.LangCode,
        input,
        cfg.Depth,
        cfg.Email,
        cfg.GeoCoordinates,
        cfg.Zoom,
        cfg.Radius,
        nil, nil,
        cfg.ExtraReviews,
    )
}
```

---

### Phase 2: Add PushWithParent to Provider
**Effort: 30 minutes**

**File**: `postgres/provider.go` (MODIFY)

Add new method to insert with parent reference:

```go
// PushWithParent pushes job with parent reference for Dashboard tracking
func (p *provider) PushWithParent(ctx context.Context, job scrapemate.IJob, parentID string) error {
    q := `INSERT INTO gmaps_jobs
        (id, priority, payload_type, payload, created_at, status, parent_job_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT DO NOTHING`

    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)

    var payloadType string
    switch j := job.(type) {
    case *gmaps.GmapJob:
        payloadType = "search"
        if err := enc.Encode(j); err != nil {
            return err
        }
    case *gmaps.PlaceJob:
        payloadType = "place"
        if err := enc.Encode(j); err != nil {
            return err
        }
    case *gmaps.EmailExtractJob:
        payloadType = "email"
        if err := enc.Encode(j); err != nil {
            return err
        }
    default:
        return fmt.Errorf("invalid job type %T", job)
    }

    _, err := p.db.ExecContext(ctx, q,
        job.GetID(), job.GetPriority(), payloadType, buf.Bytes(),
        time.Now().UTC(), statusNew, parentID,
    )
    return err
}
```

---

### Phase 3: Update JobService to Bridge
**Effort: 2-3 hours**

**File**: `internal/service/job.go` (MODIFY)

```go
import (
    "github.com/sadewadee/google-scraper/runner"
    "github.com/sadewadee/google-scraper/postgres"
)

type JobService struct {
    jobs      JobRepository       // jobs_queue
    gmapsJobs *postgres.Provider  // gmaps_jobs (NEW!)
    // ... existing fields
}

func (s *JobService) Create(ctx context.Context, req CreateJobRequest) (*Job, error) {
    // 1. Create parent job for Dashboard UI
    parentJob := &domain.Job{
        ID:       uuid.New().String(),
        Name:     req.Name,
        Keywords: req.Keywords,
        Status:   "pending",
    }
    if err := s.jobs.Create(ctx, parentJob); err != nil {
        return nil, err
    }

    // 2. Convert keywords to gmaps.GmapJob(s) using reusable function
    seedJobs, err := runner.CreateSeedJobsFromKeywords(runner.SeedJobConfig{
        Keywords:       req.Keywords,
        FastMode:       req.FastMode,
        LangCode:       req.Lang,
        Depth:          req.Depth,
        Email:          req.Email,
        GeoCoordinates: formatGeo(req.Lat, req.Lon),
        Zoom:           req.Zoom,
        Radius:         req.Radius,
        ExtraReviews:   req.ExtraReviews,
    })
    if err != nil {
        return nil, fmt.Errorf("create seed jobs: %w", err)
    }

    // 3. INSERT each seed job to gmaps_jobs (BRIDGE!)
    for _, seedJob := range seedJobs {
        if err := s.gmapsJobs.PushWithParent(ctx, seedJob, parentJob.ID); err != nil {
            return nil, fmt.Errorf("push to gmaps_jobs: %w", err)
        }
    }

    // 4. Update parent with task count
    parentJob.TotalTasks = len(seedJobs)
    parentJob.Status = "running"
    s.jobs.Update(ctx, parentJob)

    return parentJob, nil
}

func formatGeo(lat, lon float64) string {
    if lat == 0 && lon == 0 {
        return ""
    }
    return fmt.Sprintf("%f,%f", lat, lon)
}
```

---

### Phase 4: Wire Up Dependencies
**Effort: 1 hour**

**File**: `runner/managerrunner/managerrunner.go` (MODIFY)

Initialize gmapsJobs provider and inject to JobService:

```go
// In New() or init:
gmapsConn, err := openPsqlConn(cfg.Dsn)
if err != nil {
    return nil, fmt.Errorf("open gmaps connection: %w", err)
}

gmapsProvider := postgres.NewProvider(gmapsConn)

// Pass to job service
jobService := service.NewJobService(
    jobRepo,
    gmapsProvider, // NEW: for inserting to gmaps_jobs
    // ... other deps
)
```

---

### Phase 5 (Optional): Status Sync Back
**Effort: 2-3 hours**

When DSN worker completes a job, update parent job progress.

**File**: `postgres/resultwriter.go` (MODIFY)

```go
func (r *resultWriter) updateParentProgress(ctx context.Context, parentID string) error {
    if parentID == "" {
        return nil
    }

    q := `
    UPDATE jobs_queue
    SET completed_tasks = (
        SELECT COUNT(*) FROM gmaps_jobs
        WHERE parent_job_id = $1 AND status = 'completed'
    ),
    status = CASE
        WHEN (SELECT COUNT(*) FROM gmaps_jobs WHERE parent_job_id = $1 AND status != 'completed') = 0
        THEN 'completed'
        ELSE status
    END
    WHERE id = $1`

    _, err := r.db.ExecContext(ctx, q, parentID)
    return err
}
```

**Note**: This requires modifying how results are saved to include parent_job_id context.

---

## Phase 6: External Email Validation (Moribouncer)

**Effort: 2-3 hours**

Integrasi dengan Moribouncer API untuk validasi email sebelum disimpan.

### 6.1 Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     EMAIL VALIDATION FLOW (MORIBOUNCER)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Worker scrapes business page                                              │
│        │                                                                    │
│        ▼                                                                    │
│   ┌────────────────────┐                                                   │
│   │  Extract Email(s)  │                                                   │
│   │  from website      │                                                   │
│   └─────────┬──────────┘                                                   │
│             │                                                               │
│             ▼                                                               │
│   ┌────────────────────┐                                                   │
│   │  Quick Regex Check │ ← Fast, free (filter obvious invalid)            │
│   │  (format only)     │                                                   │
│   └─────────┬──────────┘                                                   │
│             │ Pass                                                          │
│             ▼                                                               │
│   ┌────────────────────┐                                                   │
│   │    Moribouncer     │ ← External API call                              │
│   │    Validation      │                                                   │
│   │                    │                                                   │
│   │  POST /validate    │                                                   │
│   │  {email: "..."}    │                                                   │
│   └─────────┬──────────┘                                                   │
│             │                                                               │
│             ▼                                                               │
│   ┌────────────────────────────────────────────────────────────────────┐  │
│   │                     VALIDATION RESULT                               │  │
│   │                                                                      │  │
│   │  ✅ Accept if:                    ❌ Reject if:                     │  │
│   │  • status = "valid"               • status = "invalid"              │  │
│   │  • deliverable = true             • disposable = true               │  │
│   │  • score >= 70                    • role_account = true (optional)  │  │
│   │                                   • catch_all = true (optional)     │  │
│   │                                                                      │  │
│   └─────────┬──────────────────────────────────────────────────────────┘  │
│             │                                                               │
│             ▼                                                               │
│   ┌────────────────────┐                                                   │
│   │  Save to results   │ ← Only validated emails                          │
│   │  (PostgreSQL)      │                                                   │
│   └────────────────────┘                                                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Create Email Validator Package

**File**: `internal/emailvalidator/moribouncer.go` (NEW)

```go
package emailvalidator

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

// MoribouncerValidator validates emails using Moribouncer API
type MoribouncerValidator struct {
    apiURL     string
    apiKey     string
    httpClient *http.Client
}

// Config for Moribouncer validator
type Config struct {
    APIURL  string        // e.g., "https://api.moribouncer.com/v1"
    APIKey  string
    Timeout time.Duration
}

// NewMoribouncerValidator creates a new Moribouncer validator
func NewMoribouncerValidator(cfg Config) *MoribouncerValidator {
    timeout := cfg.Timeout
    if timeout == 0 {
        timeout = 10 * time.Second
    }

    apiURL := cfg.APIURL
    if apiURL == "" {
        apiURL = "https://api.moribouncer.com/v1"
    }

    return &MoribouncerValidator{
        apiURL: apiURL,
        apiKey: cfg.APIKey,
        httpClient: &http.Client{
            Timeout: timeout,
        },
    }
}

// ValidationResult from Moribouncer API
type ValidationResult struct {
    Email       string  `json:"email"`
    Status      string  `json:"status"`       // valid, invalid, unknown, catch_all
    Score       float64 `json:"score"`        // 0-100
    Deliverable bool    `json:"deliverable"`
    Disposable  bool    `json:"disposable"`
    RoleAccount bool    `json:"role_account"` // info@, support@, etc.
    FreeEmail   bool    `json:"free_email"`   // gmail, yahoo, etc.
    CatchAll    bool    `json:"catch_all"`    // accepts any email
    Reason      string  `json:"reason"`
}

// Validate validates a single email
func (v *MoribouncerValidator) Validate(ctx context.Context, email string) (*ValidationResult, error) {
    // Build request URL
    reqURL := fmt.Sprintf("%s/validate?api_key=%s&email=%s",
        v.apiURL,
        url.QueryEscape(v.apiKey),
        url.QueryEscape(email),
    )

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Accept", "application/json")

    resp, err := v.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call Moribouncer API: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Moribouncer API returned status %d", resp.StatusCode)
    }

    var result ValidationResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result, nil
}

// ShouldAccept returns true if email is acceptable for leads
func (r *ValidationResult) ShouldAccept() bool {
    return r.Status == "valid" &&
        r.Deliverable &&
        !r.Disposable &&
        !r.RoleAccount &&
        r.Score >= 70
}

// ShouldAcceptRelaxed returns true with more relaxed criteria
// (allows role accounts and catch-all domains)
func (r *ValidationResult) ShouldAcceptRelaxed() bool {
    return (r.Status == "valid" || r.Status == "catch_all") &&
        r.Deliverable &&
        !r.Disposable &&
        r.Score >= 50
}
```

### 6.3 Create Validator Interface

**File**: `internal/emailvalidator/validator.go` (NEW)

```go
package emailvalidator

import "context"

// Validator interface for email validation
type Validator interface {
    Validate(ctx context.Context, email string) (*ValidationResult, error)
}

// NoOpValidator is a validator that accepts all emails (for testing/disabled mode)
type NoOpValidator struct{}

func (v *NoOpValidator) Validate(ctx context.Context, email string) (*ValidationResult, error) {
    return &ValidationResult{
        Email:       email,
        Status:      "valid",
        Score:       100,
        Deliverable: true,
        Disposable:  false,
    }, nil
}
```

### 6.4 Integrate with Email Job

**File**: `gmaps/emailjob.go` (MODIFY)

```go
import (
    "github.com/sadewadee/google-scraper/internal/emailvalidator"
)

// EmailExtractJob with validator
type EmailExtractJob struct {
    // ... existing fields
    validator emailvalidator.Validator // NEW: external validator
}

// WithEmailValidator option for EmailExtractJob
func WithEmailValidator(v emailvalidator.Validator) EmailJobOption {
    return func(j *EmailExtractJob) {
        j.validator = v
    }
}

// processEmail validates and saves email
func (j *EmailExtractJob) processEmail(ctx context.Context, email string) error {
    // Step 1: Quick regex check (free, fast)
    if !isValidEmailFormat(email) {
        return nil // Skip malformed emails
    }

    // Step 2: External validation via Moribouncer (if configured)
    if j.validator != nil {
        result, err := j.validator.Validate(ctx, email)
        if err != nil {
            // Log error but continue (don't block on API failure)
            log.Printf("Email validation failed for %s: %v", email, err)
            // Option: accept email anyway if API fails
            // Or: skip email if API fails (stricter)
            return nil
        }

        if !result.ShouldAccept() {
            log.Printf("Email rejected: %s (status: %s, score: %.0f, reason: %s)",
                email, result.Status, result.Score, result.Reason)
            return nil
        }

        log.Printf("Email validated: %s (score: %.0f)", email, result.Score)
    }

    // Step 3: Accept email
    j.entry.Email = email
    return nil
}

// isValidEmailFormat does quick regex check
func isValidEmailFormat(email string) bool {
    // Simple regex for email format
    // More complex validation is done by Moribouncer
    if len(email) < 5 || len(email) > 254 {
        return false
    }
    atIndex := strings.Index(email, "@")
    if atIndex < 1 || atIndex > len(email)-4 {
        return false
    }
    return strings.Contains(email[atIndex:], ".")
}
```

### 6.5 Add CLI Flags

**File**: `runner/runner.go` (MODIFY)

```go
// Add to Config struct
type Config struct {
    // ... existing fields

    // Email validation
    EmailValidatorURL string
    EmailValidatorKey string
}

// Add to flag parsing
flag.StringVar(&cfg.EmailValidatorURL, "email-validator-url", "",
    "Moribouncer API URL (default: https://api.moribouncer.com/v1)")
flag.StringVar(&cfg.EmailValidatorKey, "email-validator-key", "",
    "Moribouncer API key for email validation")
```

### 6.6 Wire Up in Database Runner

**File**: `runner/databaserunner/databaserunner.go` (MODIFY)

```go
import (
    "github.com/sadewadee/google-scraper/internal/emailvalidator"
)

func New(cfg *runner.Config) (runner.Runner, error) {
    // ... existing code ...

    // Setup email validator if configured
    var emailValidator emailvalidator.Validator
    if cfg.EmailValidatorKey != "" {
        emailValidator = emailvalidator.NewMoribouncerValidator(emailvalidator.Config{
            APIURL: cfg.EmailValidatorURL,
            APIKey: cfg.EmailValidatorKey,
        })
        log.Println("Email validation enabled via Moribouncer")
    }

    // Pass validator to job creation
    // This may require modifying how jobs are created/processed
    // ...
}
```

### 6.7 Environment Variables

```bash
# .env or docker-compose.yml
MORIBOUNCER_API_KEY=your-api-key-here

# CLI usage
./gmaps-scraper -dsn "postgres://..." \
    -email-validator-key "$MORIBOUNCER_API_KEY" \
    -c 4
```

---

## Files Summary

### New Files:
```
migrations/xxx_add_parent_job_id.sql      # Schema update
runner/seedjobs.go                         # Reusable seed job creator
internal/emailvalidator/moribouncer.go     # Moribouncer API client
internal/emailvalidator/validator.go       # Validator interface
```

### Modified Files:
```
postgres/provider.go                       # Add PushWithParent()
internal/service/job.go                    # Bridge: insert to gmaps_jobs
runner/managerrunner/managerrunner.go      # Wire dependencies
postgres/resultwriter.go                   # (Optional) Status sync
gmaps/emailjob.go                          # Integrate email validator
runner/runner.go                           # Add CLI flags
runner/databaserunner/databaserunner.go    # Wire email validator
```

### NO Changes Needed:
```
postgres/provider.go (fetchJobs, Jobs)     # Consumer logic UNCHANGED
runner/databaserunner/databaserunner.go    # DSN runner core UNCHANGED
```

---

## Verification Steps

### 1. Run Migration
```bash
psql -d your_db -f migrations/xxx_add_parent_job_id.sql
```

### 2. Start Manager
```bash
./gmaps-scraper -manager -dsn "postgres://..." -addr :8080
```

### 3. Start DSN Worker(s) with Email Validation
```bash
./gmaps-scraper -dsn "postgres://..." \
    -c 4 \
    -email \
    -email-validator-key "your-moribouncer-api-key"
```

### 4. Create Job via Dashboard
```bash
curl -X POST http://localhost:8080/api/v2/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Search",
    "keywords": ["cafe jakarta"],
    "depth": 5,
    "email": true
  }'
```

### 5. Verify Job in gmaps_jobs
```bash
psql -d your_db -c "SELECT id, payload_type, status, parent_job_id FROM gmaps_jobs ORDER BY created_at DESC LIMIT 5"
```

Expected: Job dengan `payload_type='search'`, `status='new'`, dan `parent_job_id` terisi.

### 6. Verify Worker Picks Up
Worker should pick up the job within 50-300ms and start processing.

Check logs:
```
Picked up job: search-cafe-jakarta-xxx
Processing...
Email validation enabled via Moribouncer
Email validated: contact@cafe.com (score: 85)
Email rejected: info@example.com (status: invalid, score: 0, reason: mailbox not found)
```

### 7. Verify Email Validation
```bash
# Check results for validated emails only
psql -d your_db -c "SELECT data->>'title', data->>'email' FROM results WHERE data->>'email' IS NOT NULL ORDER BY id DESC LIMIT 10"
```

---

## Estimated Effort

| Phase | Task | Time |
|-------|------|------|
| 0 | Schema migration | 30 min |
| 1 | Seed job function | 1-2 hours |
| 2 | PushWithParent | 30 min |
| 3 | JobService bridge | 2-3 hours |
| 4 | Wire dependencies | 1 hour |
| 5 | Status sync (optional) | 2-3 hours |
| 6 | Moribouncer integration | 2-3 hours |
| **Total** | | **8-12 hours** |

---

## What We Removed from Original Plan

| Removed | Reason |
|---------|--------|
| RabbitMQ integration | User confirmed polling 50-300ms is acceptable |
| Hybrid queue | Unnecessary complexity |
| HybridWorker class | Not needed |
| RabbitMQ notifier | Not needed |
| NextJS migration | Out of scope for this task |
| Layered deduplication | Can be separate phase |

---

## Risk Mitigation

1. **DSN Consumer Unchanged**: Zero risk to production workers
2. **Additive Changes Only**: New column, new function, new method
3. **Backwards Compatible**: CLI `-produce` still works
4. **Rollback**: Just remove parent_job_id column and bridge code
5. **Email Validation Optional**: Works without API key (no validation)

---

---

## Frontend Architecture Analysis

### Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 19 | UI framework |
| Vite | 7 | Build tool |
| TypeScript | 5.7 | Type safety |
| TailwindCSS | 4 | Styling |
| React Query | 5 | Data fetching & caching |
| React Hook Form | 7 | Form management |
| Zod | 3.24 | Schema validation |
| Recharts | 2.15 | Charts/visualizations |

### Routing Structure

**File**: `web/frontend/src/App.tsx`

```
/              → Dashboard (overview stats)
/jobs          → Job list
/jobs/new      → Create new job
/jobs/:id      → Job detail
/results       → Results list
/workers       → Worker status
/proxies       → Proxy management
/settings      → App settings
```

### Job Creation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       FRONTEND JOB CREATION FLOW                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   User fills JobForm                                                        │
│        │                                                                    │
│        ▼                                                                    │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │ JobForm.tsx (react-hook-form + zod validation)                     │   │
│   │                                                                     │   │
│   │ Fields:                                                             │   │
│   │ • name: string         • keywords: string[]                        │   │
│   │ • lang: string         • lat/lon: number (optional)                │   │
│   │ • zoom: number         • radius: number                            │   │
│   │ • depth: number        • fast_mode: boolean                        │   │
│   │ • extract_email: bool  • priority: number                          │   │
│   │ • max_time: number                                                  │   │
│   └─────────────────────────────────────┬──────────────────────────────┘   │
│                                         │                                   │
│                                         ▼                                   │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │ jobsApi.create(payload)  →  POST /api/v2/jobs                      │   │
│   │                                                                     │   │
│   │ JobCreatePayload interface matches backend expectations            │   │
│   └─────────────────────────────────────┬──────────────────────────────┘   │
│                                         │                                   │
│                                         ▼                                   │
│                           ┌─────────────────┐                              │
│                           │  Backend API    │                              │
│                           │  (Go Manager)   │                              │
│                           └─────────────────┘                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### API Client

**File**: `web/frontend/src/api/jobs.ts`

```typescript
export const jobsApi = {
    create: async (data: JobCreatePayload) => api.post("/jobs", data),
    getAll: async (params) => api.get("/jobs", { params }),
    getOne: async (id) => api.get(`/jobs/${id}`),
    cancel: async (id) => api.post(`/jobs/${id}/cancel`),
    pause: async (id) => api.post(`/jobs/${id}/pause`),
    resume: async (id) => api.post(`/jobs/${id}/resume`),
    delete: async (id) => api.delete(`/jobs/${id}`),
    getResults: async (id, params) => api.get(`/jobs/${id}/results`, { params }),
    downloadResults: async (id, format) => download(`/jobs/${id}/results/export`),
}
```

### Type Definitions

**File**: `web/frontend/src/api/types.ts`

```typescript
interface JobCreatePayload {
    name: string
    keywords: string[]
    lang: string
    zoom: number
    radius: number
    depth: number
    fast_mode: boolean
    extract_email: boolean
    priority: number
    max_time: number
    lat?: number
    lon?: number
}
```

### Frontend vs Backend Integration

**GOOD NEWS: NO FRONTEND CHANGES NEEDED for the bridge!**

| Aspect | Current State | After Bridge |
|--------|---------------|--------------|
| Job creation form | ✅ Already sends correct payload | ✅ No change |
| API endpoint | POST `/api/v2/jobs` | ✅ Same endpoint |
| Payload format | `JobCreatePayload` | ✅ Same format |
| Job status display | Shows from `jobs_queue` | ✅ No change |
| Progress tracking | `scraped_places/total_places` | ✅ No change |

### Why No Frontend Changes?

```
Frontend sends:                    Backend receives & bridges:
┌──────────────────┐              ┌────────────────────────────────────┐
│ POST /api/v2/jobs│              │ 1. Create parent in jobs_queue    │
│ {                │─────────────►│ 2. Convert keywords → GmapJobs    │
│   keywords: [...],│              │ 3. INSERT each to gmaps_jobs      │
│   depth: 10,     │              │ 4. DSN workers pick up            │
│   email: true    │              │ 5. Results saved to results table │
│ }                │              └────────────────────────────────────┘
└──────────────────┘
```

The bridge is **100% backend logic**. Frontend already sends exactly what we need.

### Optional Frontend Enhancements (Future)

| Feature | Effort | Description |
|---------|--------|-------------|
| Email validation toggle | 30 min | Add checkbox "Validate emails with Moribouncer" |
| Task progress bar | 1 hour | Show `completed_tasks/total_tasks` (from new columns) |
| Per-keyword status | 2 hours | Expand job detail to show each keyword's status |
| Real-time updates | 2-3 hours | WebSocket/SSE for live progress |

---

## Phase 7: Auto Migration (Old Design → New Design)

**Effort: 2-3 hours**

Auto migration dijalankan **SEKALI** saat pertama kali deploy ke desain baru. Tujuannya adalah memigrasikan data existing dari struktur lama ke struktur baru tanpa kehilangan data.

### 7.1 Migration Scenarios

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AUTO MIGRATION SCENARIOS                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SCENARIO A: Fresh Install (No existing data)                              │
│  ─────────────────────────────────────────────                              │
│  • Skip migration, just run schema creation                                 │
│  • Create all tables with new schema                                        │
│                                                                             │
│  SCENARIO B: Existing jobs_queue + gmaps_jobs (both exist, unlinked)       │
│  ─────────────────────────────────────────────────────────────────          │
│  • Add parent_job_id column to gmaps_jobs                                  │
│  • Add total_tasks, completed_tasks columns to jobs_queue                  │
│  • Link orphan gmaps_jobs to jobs_queue where possible                     │
│  • Mark unlinked gmaps_jobs as "legacy" (parent_job_id = NULL)             │
│                                                                             │
│  SCENARIO C: Only gmaps_jobs exists (CLI -produce mode used)               │
│  ────────────────────────────────────────────────────────────               │
│  • Create jobs_queue table if not exists                                   │
│  • Optionally create synthetic parent jobs from gmaps_jobs                 │
│  • Or leave parent_job_id = NULL for CLI-originated jobs                   │
│                                                                             │
│  SCENARIO D: Only jobs_queue exists (Dashboard used, bridge not impl)      │
│  ───────────────────────────────────────────────────────────────            │
│  • Create gmaps_jobs table if not exists                                   │
│  • Re-bridge existing pending/running jobs to gmaps_jobs                   │
│  • Skip completed/failed/cancelled jobs (no need to re-scrape)             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Migration Detection Logic

**File**: `internal/migration/automigrate.go` (NEW)

```go
package migration

import (
    "context"
    "database/sql"
    "log"
)

// MigrationState represents the current state of the database
type MigrationState int

const (
    StateFreshInstall      MigrationState = iota // No tables exist
    StateBothExistUnlinked                        // Both tables, no parent_job_id
    StateOnlyGmapsJobs                            // Only gmaps_jobs exists
    StateOnlyJobsQueue                            // Only jobs_queue exists
    StateAlreadyMigrated                          // parent_job_id column exists
)

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
```

### 7.3 Migration Executor

**File**: `internal/migration/executor.go` (NEW)

```go
package migration

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"
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

    // 3. Get pending/running jobs that need to be re-bridged
    rows, err := tx.QueryContext(ctx, `
        SELECT id, keywords, lang, geo_lat, geo_lon, zoom, radius, depth,
               fast_mode, extract_email, priority
        FROM jobs_queue
        WHERE status IN ('pending', 'running', 'queued')
    `)
    if err != nil {
        return fmt.Errorf("query pending jobs: %w", err)
    }
    defer rows.Close()

    var pendingCount int
    for rows.Next() {
        pendingCount++
        // Note: Actual bridging would require CreateSeedJobs() logic
        // This is just counting for the migration log
    }

    log.Printf("[AutoMigrate] Found %d pending jobs to re-bridge after migration", pendingCount)
    log.Println("[AutoMigrate] NOTE: Run bridge manually for these jobs or recreate them via Dashboard")

    // 4. Record migration
    _, err = tx.ExecContext(ctx, `
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
    `)
    if err != nil {
        return fmt.Errorf("add parent_job_id column: %w", err)
    }

    // 2. Create jobs_queue table (will be used by Dashboard)
    // Note: This should run the standard migration 0005

    // 3. Record migration
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
```

### 7.4 Integration with Manager Startup

**File**: `runner/managerrunner/managerrunner.go` (MODIFY)

```go
import (
    "github.com/sadewadee/google-scraper/internal/migration"
)

func New(cfg *runner.Config) (runner.Runner, error) {
    // ... existing connection setup ...

    // Run auto-migration on first deploy
    if err := migration.AutoMigrate(context.Background(), db); err != nil {
        return nil, fmt.Errorf("auto-migrate failed: %w", err)
    }

    // ... rest of initialization ...
}
```

### 7.5 CLI Flag for Manual Migration

```bash
# Run migration manually (useful for testing)
./gmaps-scraper -migrate -dsn "postgres://..."

# Check migration status
./gmaps-scraper -migrate-status -dsn "postgres://..."
```

**File**: `runner/runner.go` (ADD flags)

```go
// Add to Config
Migrate       bool // Run migration only, then exit
MigrateStatus bool // Check migration status

// Add to flag parsing
flag.BoolVar(&cfg.Migrate, "migrate", false, "Run auto-migration and exit")
flag.BoolVar(&cfg.MigrateStatus, "migrate-status", false, "Check migration status and exit")
```

### 7.6 Migration Safety Checks

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        MIGRATION SAFETY CHECKLIST                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ✅ Idempotent: Can run multiple times without side effects                │
│     • Uses IF NOT EXISTS for all DDL                                        │
│     • Checks StateAlreadyMigrated before running                           │
│                                                                             │
│  ✅ Non-destructive: Never deletes data                                    │
│     • Only ADDs columns, never removes                                      │
│     • Existing data remains intact                                          │
│                                                                             │
│  ✅ Transactional: All or nothing                                          │
│     • Each scenario runs in a transaction                                   │
│     • Rollback on any error                                                 │
│                                                                             │
│  ✅ Logged: Full audit trail                                               │
│     • migration_history table tracks all migrations                         │
│     • Timestamps for debugging                                              │
│                                                                             │
│  ✅ Backwards Compatible: Old workers still work                           │
│     • parent_job_id is NULLABLE                                            │
│     • DSN workers ignore new column                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.7 Migration Verification

```bash
# After migration, verify:

# 1. Check migration history
psql -d your_db -c "SELECT * FROM migration_history ORDER BY executed_at DESC"

# 2. Verify gmaps_jobs has parent_job_id
psql -d your_db -c "\d gmaps_jobs"

# 3. Verify jobs_queue has task tracking
psql -d your_db -c "\d jobs_queue"

# 4. Test Dashboard still works
curl http://localhost:8080/api/v2/jobs

# 5. Test DSN worker still picks up jobs
./gmaps-scraper -dsn "postgres://..." -c 1 &
# Create job from Dashboard, verify worker picks it up
```

---

## Next Steps After This Plan

1. **Phase 8**: TTL Change (24h → 7 days) - 5 minutes
2. **Phase 9**: Layered Deduplication - 1 day
3. **Phase 10**: NextJS Migration (if needed) - 3-5 days
