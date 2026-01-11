-- Migration 0001: Manager Worker Architecture
-- Creates all required tables for the manager/worker system

BEGIN;

-- =====================================================
-- Results Table (create if not exists)
-- =====================================================
CREATE TABLE IF NOT EXISTS results (
    id BIGSERIAL PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =====================================================
-- Jobs Queue Table
-- For manager/worker architecture
-- =====================================================
CREATE TABLE IF NOT EXISTS jobs_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 0,

    -- Search configuration
    keywords TEXT[] NOT NULL,
    lang TEXT NOT NULL DEFAULT 'en',
    geo_lat DOUBLE PRECISION,
    geo_lon DOUBLE PRECISION,
    zoom INT DEFAULT 15,
    radius INT DEFAULT 10000,
    depth INT NOT NULL DEFAULT 10,
    fast_mode BOOLEAN DEFAULT FALSE,
    extract_email BOOLEAN DEFAULT FALSE,
    max_time INTERVAL DEFAULT '10 minutes',
    proxies TEXT[],

    -- Progress tracking
    total_places INT DEFAULT 0,
    scraped_places INT DEFAULT 0,
    failed_places INT DEFAULT 0,

    -- Worker assignment
    worker_id TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Error info
    error_message TEXT,

    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('pending', 'queued', 'running', 'paused', 'completed', 'failed', 'cancelled'))
);

-- Indexes for jobs_queue
CREATE INDEX IF NOT EXISTS idx_jobs_queue_status ON jobs_queue(status);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_created_at ON jobs_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_worker_id ON jobs_queue(worker_id);
CREATE INDEX IF NOT EXISTS idx_jobs_queue_pending ON jobs_queue(status, priority DESC, created_at ASC)
    WHERE status = 'pending';

-- =====================================================
-- Workers Table
-- Track worker status and heartbeat
-- =====================================================
CREATE TABLE IF NOT EXISTS workers (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    current_job_id UUID REFERENCES jobs_queue(id) ON DELETE SET NULL,

    -- Stats
    jobs_completed INT DEFAULT 0,
    places_scraped INT DEFAULT 0,

    -- Heartbeat
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_worker_status CHECK (status IN ('idle', 'busy', 'offline'))
);

-- Index for worker heartbeat cleanup
CREATE INDEX IF NOT EXISTS idx_workers_heartbeat ON workers(last_heartbeat);

-- =====================================================
-- Results Table Enhancement
-- Add job reference (only if column doesn't exist)
-- =====================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'results' AND column_name = 'job_id'
    ) THEN
        ALTER TABLE results ADD COLUMN job_id UUID REFERENCES jobs_queue(id) ON DELETE CASCADE;
        CREATE INDEX idx_results_job_id ON results(job_id);
    END IF;
END $$;

-- =====================================================
-- Function to update updated_at timestamp
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for jobs_queue
DROP TRIGGER IF EXISTS update_jobs_queue_updated_at ON jobs_queue;
CREATE TRIGGER update_jobs_queue_updated_at
    BEFORE UPDATE ON jobs_queue
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMIT;
