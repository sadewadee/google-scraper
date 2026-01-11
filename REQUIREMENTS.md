# Frontend & Backend Requirements Analysis

## Current State Analysis

### Current Frontend (`web/static/`)

| Component | Technology | Issues |
|-----------|------------|--------|
| Templates | Go `html/template` | Server-rendered, limited interactivity |
| Styling | Vanilla CSS (358 lines) | No dark mode, basic responsive |
| Interactivity | HTMX (CDN) | 10s polling, inline JS |
| Forms | HTML forms | Server-side validation only |

**Files:**
- `templates/index.html` - Main page (160 lines)
- `templates/job_row.html` - Single job row
- `templates/job_rows.html` - Job list
- `templates/redoc.html` - API docs
- `css/main.css` - Styles
- `spec/spec.yaml` - OpenAPI spec

**Current Features:**
- ✅ Create job (form submit)
- ✅ List jobs (polling every 10s)
- ✅ Delete job (with confirm)
- ✅ Download results (CSV)
- ❌ No progress tracking
- ❌ No pause/resume
- ❌ No worker visibility
- ❌ No real-time updates

---

### Current Backend (`web/`)

| Component | Technology | Issues |
|-----------|------------|--------|
| HTTP Server | `net/http` | No middleware framework |
| Database | SQLite (`web/sqlite/`) | Single instance, no scale |
| API | REST (partial) | Mixed HTML/JSON responses |
| Validation | Manual in handlers | Duplicated logic |

**Current API Endpoints:**
```
GET  /                      → HTML index page
POST /scrape                → Create job (HTML form)
GET  /jobs                  → List jobs (HTML fragment)
GET  /download?id=          → Download CSV
GET  /delete?id=            → Delete job

GET  /api/v1/jobs           → List jobs (JSON)
POST /api/v1/jobs           → Create job (JSON)
GET  /api/v1/jobs/{id}      → Get job (JSON)
DELETE /api/v1/jobs/{id}    → Delete job (JSON)
GET  /api/v1/jobs/{id}/download → Download CSV
```

**Current Data Model (`web/job.go`):**
```go
type Job struct {
    ID     string    // UUID
    Name   string    // Job name
    Date   time.Time // Created at
    Status string    // pending, working, ok, failed
    Data   JobData   // Config
}

type JobData struct {
    Keywords []string
    Lang     string
    Zoom     int
    Lat      string
    Lon      string
    FastMode bool
    Radius   int
    Depth    int
    Email    bool
    MaxTime  time.Duration
    Proxies  []string
}
```

**Status Values:**
- `pending` - Created, waiting to process
- `working` - Currently scraping
- `ok` - Completed successfully
- `failed` - Error occurred

---

## New Architecture Requirements

### Frontend Requirements

#### 1. Technology Stack
| Choice | Reasoning |
|--------|-----------|
| **React 18** | Component-based, large ecosystem |
| **TypeScript** | Type safety, better DX |
| **TailwindCSS** | Rapid styling, dark mode built-in |
| **React Query** | Server state, caching, polling |
| **Vite** | Fast builds, HMR |
| **React Router** | Client-side routing |

#### 2. Pages & Components

```
src/
├── pages/
│   ├── Dashboard.tsx       # Main page - job list, stats
│   ├── JobCreate.tsx       # Create new job form
│   ├── JobDetail.tsx       # Job detail + results
│   └── Workers.tsx         # Worker status dashboard
├── components/
│   ├── Layout/
│   │   ├── Header.tsx      # Logo, nav, theme toggle
│   │   ├── Sidebar.tsx     # Navigation menu
│   │   └── Footer.tsx
│   ├── Job/
│   │   ├── JobForm.tsx     # Create/edit job form
│   │   ├── JobTable.tsx    # Desktop job list
│   │   ├── JobCard.tsx     # Mobile job card
│   │   ├── JobStatus.tsx   # Status badge
│   │   └── JobProgress.tsx # Progress bar
│   ├── Results/
│   │   ├── ResultsTable.tsx
│   │   ├── ResultsExport.tsx
│   │   └── ResultsPreview.tsx
│   ├── Worker/
│   │   ├── WorkerCard.tsx
│   │   └── WorkerStats.tsx
│   └── UI/
│       ├── Button.tsx
│       ├── Input.tsx
│       ├── Modal.tsx
│       ├── Toast.tsx
│       ├── Skeleton.tsx
│       └── EmptyState.tsx
├── hooks/
│   ├── useJobs.ts          # React Query - jobs CRUD
│   ├── useWorkers.ts       # React Query - workers
│   ├── useResults.ts       # React Query - results
│   └── useTheme.ts         # Dark mode toggle
├── api/
│   ├── client.ts           # Axios/fetch wrapper
│   ├── jobs.ts             # Job API calls
│   ├── workers.ts          # Worker API calls
│   └── results.ts          # Results API calls
├── types/
│   ├── job.ts
│   ├── worker.ts
│   └── result.ts
└── App.tsx
```

#### 3. UI Features Required

| Feature | Priority | Description |
|---------|----------|-------------|
| Job CRUD | P0 | Create, view, delete jobs |
| Job List | P0 | Table with sorting, filtering |
| Real-time Updates | P0 | Polling 2-3s or WebSocket |
| Progress Tracking | P0 | Progress bar per job |
| Results Preview | P1 | Preview first 10 rows |
| Results Export | P1 | CSV/JSON download |
| Worker Dashboard | P1 | Worker status, stats |
| Dark Mode | P1 | Theme toggle |
| Responsive | P1 | Mobile-first design |
| Toast Notifications | P1 | Success/error feedback |
| Pause/Resume | P2 | Pause running job |
| Bulk Actions | P2 | Select multiple, delete |
| Search/Filter | P2 | Filter jobs by status |

#### 4. Form Fields (Job Create)

**Basic (Always Visible):**
- Job Name (text, required)
- Keywords (textarea, required, one per line)
- Language (select, default: en)

**Location Settings (Collapsible):**
- Latitude (number)
- Longitude (number)
- Zoom (number, 1-21, default: 15)
- Radius (number, meters)

**Advanced Options (Collapsible):**
- Depth (number, 1-100, default: 10)
- Fast Mode (checkbox)
- Extract Emails (checkbox)
- Max Time (duration, default: 10m)

**Proxies (Collapsible):**
- Proxy List (textarea, one per line)

---

### Backend Requirements

#### 1. Technology Stack (Keep Go)
| Component | Current | New |
|-----------|---------|-----|
| HTTP Router | `net/http` | `chi` or `echo` |
| Database | SQLite | PostgreSQL |
| Migrations | Manual | `golang-migrate` |
| Validation | Manual | `go-playground/validator` |
| Config | CLI flags | `viper` + env vars |

#### 2. New Database Schema

```sql
-- Jobs queue (replaces SQLite jobs table)
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    priority INT DEFAULT 0,

    -- Config
    config JSONB NOT NULL,

    -- Progress tracking
    total_places INT DEFAULT 0,
    scraped_places INT DEFAULT 0,
    failed_places INT DEFAULT 0,

    -- Worker assignment
    worker_id TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Error info
    error_message TEXT
);

-- Indexes
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_worker_id ON jobs(worker_id);

-- Workers heartbeat
CREATE TABLE workers (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    current_job_id UUID REFERENCES jobs(id),

    -- Stats
    jobs_completed INT DEFAULT 0,
    places_scraped INT DEFAULT 0,

    -- Heartbeat
    last_heartbeat TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Results (enhanced)
CREATE TABLE results (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_results_job_id ON results(job_id);

-- Job status enum
-- pending, queued, running, paused, completed, failed, cancelled
```

#### 3. New API Endpoints

```yaml
# Jobs
GET    /api/v1/jobs                    # List jobs (paginated)
POST   /api/v1/jobs                    # Create job
GET    /api/v1/jobs/:id                # Get job detail
DELETE /api/v1/jobs/:id                # Delete job
PATCH  /api/v1/jobs/:id                # Update job (pause/resume/cancel)
GET    /api/v1/jobs/:id/results        # Get job results (paginated)
GET    /api/v1/jobs/:id/results/export # Export results (CSV/JSON)

# Workers
GET    /api/v1/workers                 # List workers
GET    /api/v1/workers/:id             # Get worker detail

# Stats
GET    /api/v1/stats                   # Dashboard stats

# Health
GET    /health                         # Health check
GET    /ready                          # Readiness check
```

#### 4. API Response Format

```typescript
// Standard response wrapper
interface ApiResponse<T> {
    data: T;
    meta?: {
        total: number;
        page: number;
        per_page: number;
    };
}

// Error response
interface ApiError {
    error: {
        code: string;
        message: string;
        details?: Record<string, string>;
    };
}

// Job
interface Job {
    id: string;
    name: string;
    status: 'pending' | 'queued' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled';
    config: JobConfig;
    progress: {
        total: number;
        scraped: number;
        failed: number;
        percentage: number;
    };
    worker_id?: string;
    created_at: string;
    started_at?: string;
    completed_at?: string;
    error_message?: string;
}

// Worker
interface Worker {
    id: string;
    hostname: string;
    status: 'idle' | 'busy' | 'offline';
    current_job_id?: string;
    jobs_completed: number;
    places_scraped: number;
    last_heartbeat: string;
}

// Stats
interface Stats {
    jobs: {
        total: number;
        pending: number;
        running: number;
        completed: number;
        failed: number;
    };
    workers: {
        total: number;
        online: number;
        busy: number;
    };
    places: {
        total_scraped: number;
        today: number;
    };
}
```

#### 5. Backend Architecture

```
cmd/
├── server/
│   └── main.go           # Web UI server (manager mode)
└── worker/
    └── main.go           # Scraper worker

internal/
├── api/
│   ├── handlers/
│   │   ├── jobs.go
│   │   ├── workers.go
│   │   └── stats.go
│   ├── middleware/
│   │   ├── cors.go
│   │   ├── logging.go
│   │   └── recovery.go
│   └── router.go
├── domain/
│   ├── job.go
│   ├── worker.go
│   └── result.go
├── repository/
│   ├── postgres/
│   │   ├── job.go
│   │   ├── worker.go
│   │   └── result.go
│   └── interfaces.go
├── service/
│   ├── job.go
│   └── worker.go
└── config/
    └── config.go
```

---

## Migration Path

### Phase 1: Backend Changes
1. Add PostgreSQL support to existing codebase
2. Create new migration files
3. Add new API endpoints alongside existing
4. Add worker heartbeat system
5. Test with existing frontend

### Phase 2: Frontend Development
1. Setup React + Vite project
2. Build core components (Layout, JobTable, JobForm)
3. Integrate with new API
4. Add real-time updates
5. Add dark mode

### Phase 3: Integration
1. Build React app for production
2. Embed in Go binary
3. Serve SPA from Go server
4. Remove old templates

### Phase 4: Worker Separation
1. Create separate worker binary
2. Add manager-only mode to server
3. Update docker-compose
4. Test distributed setup

---

## File Changes Summary

### Files to Create
```
web/frontend/                    # New React app
├── src/
├── package.json
├── vite.config.ts
└── tailwind.config.js

internal/                        # Refactored backend
├── api/
├── domain/
├── repository/
└── service/

scripts/migrations/
├── 0002_enhance_jobs_table.up.sql
├── 0002_enhance_jobs_table.down.sql
├── 0003_create_workers_table.up.sql
└── 0003_create_workers_table.down.sql
```

### Files to Modify
```
web/web.go                       # Add new endpoints
web/job.go                       # Enhance Job struct
postgres/resultwriter.go         # Add job_id support
runner/runner.go                 # Add manager-only flag
docker-compose.yml               # Separate web/worker
```

### Files to Remove (Phase 3)
```
web/static/templates/            # Replace with React SPA
web/static/css/
```

---

## Proxy System (Current Implementation)

### How Proxies Work

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Input                               │
│                                                                 │
│  CLI: -proxies "http://user:pass@proxy1:8080,socks5://proxy2"  │
│  Web UI: Textarea (one per line)                                │
│  Config: cfg.Proxies []string                                   │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     runner/runner.go                            │
│                                                                 │
│  // Parse comma-separated list                                  │
│  cfg.Proxies = strings.Split(proxies, ",")                     │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                  scrapemateapp.WithProxies()                    │
│                                                                 │
│  // Passed to scrapemate library                                │
│  opts = append(opts, scrapemateapp.WithProxies(cfg.Proxies))   │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                 Browser (Playwright/Rod)                        │
│                                                                 │
│  • All HTTP requests go through proxy                           │
│  • Automatic rotation if multiple proxies                       │
│  • Supports HTTP, HTTPS, SOCKS5                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Proxy Priority

```go
// runner/webrunner/webrunner.go:268-278
if len(w.cfg.Proxies) > 0 {
    // 1. Global config (CLI -proxies flag) - highest priority
    opts = append(opts, scrapemateapp.WithProxies(w.cfg.Proxies))
} else if len(job.Data.Proxies) > 0 {
    // 2. Per-job proxies (Web UI form) - fallback
    opts = append(opts, scrapemateapp.WithProxies(job.Data.Proxies))
}
// 3. No proxy - direct connection
```

### Supported Proxy Formats

| Type | Format | Example |
|------|--------|---------|
| HTTP | `http://[user:pass@]host:port` | `http://user:pass@proxy.com:8080` |
| HTTPS | `https://[user:pass@]host:port` | `https://user:pass@proxy.com:443` |
| SOCKS5 | `socks5://[user:pass@]host:port` | `socks5://127.0.0.1:1080` |

### Multiple Proxies (Load Balancing)

```bash
# CLI - comma separated
./gmaps-scraper -proxies "http://proxy1:8080,http://proxy2:8080,socks5://proxy3:1080"

# Web UI - one per line
http://user:pass@proxy1.example.com:8080
http://user:pass@proxy2.example.com:8080
socks5://proxy3.example.com:1080
```

Scrapemate automatically rotates between proxies for each request.

### Proxy Configuration Locations

| Location | Priority | Use Case |
|----------|----------|----------|
| CLI flag `-proxies` | 1 (highest) | Global for all jobs |
| Job config (Web UI) | 2 | Per-job proxies |
| No proxy | 3 (default) | Direct connection |

---

## Email Validation System

### 3-Tier Validation Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Email Validation Flow                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Discovered Email                                              │
│         │                                                       │
│         ▼                                                       │
│   ┌─────────────────────────────────────────────────────┐      │
│   │  Tier 1: Mordibouncer API (Primary)                 │      │
│   │  - Full SMTP verification without sending email     │      │
│   │  - Disposable detection                             │      │
│   │  - Role account detection                           │      │
│   │  - Catch-all detection                              │      │
│   │  - Gravatar check                                   │      │
│   │  - HaveIBeenPwned check                            │      │
│   └─────────────────────────────────────────────────────┘      │
│         │                                                       │
│         │ If not configured or API fails                       │
│         ▼                                                       │
│   ┌─────────────────────────────────────────────────────┐      │
│   │  Tier 2: Built-in Validation (Fallback)             │      │
│   │  - Syntax validation (regex)                        │      │
│   │  - MX record lookup (DNS)                           │      │
│   │  - Disposable domain list (local, 1000+ domains)    │      │
│   │  - Role account list (local)                        │      │
│   └─────────────────────────────────────────────────────┘      │
│         │                                                       │
│         │ If MX lookup disabled or fails                       │
│         ▼                                                       │
│   ┌─────────────────────────────────────────────────────┐      │
│   │  Tier 3: Basic Validation (Minimal)                 │      │
│   │  - Syntax validation only                           │      │
│   │  - Domain extraction                                │      │
│   └─────────────────────────────────────────────────────┘      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Tier 1: Mordibouncer (Primary)

External service providing full SMTP verification.

**API:**
```bash
POST /v0/check_email
Headers: x-mordibouncer-secret: <api-key>
Body: {"to_email": "test@gmail.com"}
```

**Response:**
```json
{
  "is_reachable": "safe",  // safe, risky, invalid, unknown
  "misc": {
    "is_disposable": false,
    "is_role_account": false
  },
  "smtp": {
    "is_deliverable": true,
    "is_catch_all": false
  }
}
```

**Configuration:**
```bash
-email-validator-url "https://mordibouncer.example.com"
-email-validator-key "your-api-key"
```

### Tier 2: Built-in (Fallback)

Local validation when Mordibouncer is not available.

**Features:**
- Syntax validation (RFC 5322)
- MX record lookup (cached)
- Disposable domain detection (1000+ domains)
- Role account detection (info@, support@, admin@, etc.)

**Configuration:**
```bash
-email-validate           # Enable validation
-email-validate-mx        # Enable MX lookup
-email-skip-role          # Exclude role accounts
-email-skip-disposable    # Exclude disposable emails
```

### Tier 3: Basic (Minimal)

Syntax-only validation as last resort.

### Confidence Mapping

| Source | Reachability | Confidence |
|--------|--------------|------------|
| Mordibouncer | safe | 1.0 |
| Mordibouncer | risky | 0.6 |
| Mordibouncer | invalid | 0.0 |
| Mordibouncer | unknown | 0.3 |
| Built-in (MX OK) | - | 0.7 |
| Built-in (MX fail) | - | 0.4 |
| Basic (syntax only) | - | 0.3 |
