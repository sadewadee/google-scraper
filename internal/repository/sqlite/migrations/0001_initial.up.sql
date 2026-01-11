-- Migration 0001: Initial SQLite Schema

-- =====================================================
-- Results Table
-- =====================================================
CREATE TABLE IF NOT EXISTS results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT, -- UUID as TEXT
    data TEXT NOT NULL, -- JSON as TEXT
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_results_job_id ON results(job_id);

-- =====================================================
-- Jobs Queue Table
-- =====================================================
CREATE TABLE IF NOT EXISTS jobs_queue (
    id TEXT PRIMARY KEY, -- UUID as TEXT
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL DEFAULT 0,

    -- Search configuration
    keywords TEXT NOT NULL, -- JSON array as TEXT
    lang TEXT NOT NULL DEFAULT 'en',
    geo_lat REAL,
    geo_lon REAL,
    zoom INTEGER DEFAULT 15,
    radius INTEGER DEFAULT 10000,
    depth INTEGER NOT NULL DEFAULT 10,
    fast_mode BOOLEAN DEFAULT 0,
    extract_email BOOLEAN DEFAULT 0,
    max_time TEXT DEFAULT '10m', -- Duration string
    proxies TEXT, -- JSON array as TEXT

    -- Progress tracking
    total_places INTEGER DEFAULT 0,
    scraped_places INTEGER DEFAULT 0,
    failed_places INTEGER DEFAULT 0,

    -- Worker assignment
    worker_id TEXT,

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    started_at TEXT,
    completed_at TEXT,

    -- Error info
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_jobs_queue_status ON jobs_queue(status);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_created_at ON jobs_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_worker_id ON jobs_queue(worker_id);

-- =====================================================
-- Workers Table
-- =====================================================
CREATE TABLE IF NOT EXISTS workers (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    current_job_id TEXT,

    -- Stats
    jobs_completed INTEGER DEFAULT 0,
    places_scraped INTEGER DEFAULT 0,

    -- Heartbeat
    last_heartbeat TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_workers_heartbeat ON workers(last_heartbeat);

-- =====================================================
-- Schema Migrations Table
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
