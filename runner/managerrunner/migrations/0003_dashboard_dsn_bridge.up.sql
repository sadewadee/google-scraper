BEGIN;

-- =====================================================
-- Dashboard → DSN Bridge Migration
-- Links gmaps_jobs back to jobs_queue for progress tracking
-- =====================================================

-- 1. Create gmaps_jobs table if it doesn't exist (Scenario D/Fresh Install)
CREATE TABLE IF NOT EXISTS gmaps_jobs (
    id TEXT PRIMARY KEY,
    priority INT DEFAULT 0,
    payload_type VARCHAR(50),
    payload BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'new',
    parent_job_id UUID
);

-- 2. Add parent_job_id if table existed but column didn't (Scenario B/C)
ALTER TABLE gmaps_jobs ADD COLUMN IF NOT EXISTS parent_job_id UUID;

-- 3. Create indexes
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_status ON gmaps_jobs(status);
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_priority ON gmaps_jobs(priority DESC);
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent ON gmaps_jobs(parent_job_id);
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent_status ON gmaps_jobs(parent_job_id, status);

-- 4. Add task tracking columns to jobs_queue
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS total_tasks INTEGER DEFAULT 0;
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS completed_tasks INTEGER DEFAULT 0;

COMMIT;
