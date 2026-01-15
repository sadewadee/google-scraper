BEGIN;

-- =====================================================
-- Dashboard → DSN Bridge Migration
-- Links gmaps_jobs back to jobs_queue for progress tracking
-- =====================================================

-- Add parent_job_id to gmaps_jobs for linking to Dashboard jobs
ALTER TABLE gmaps_jobs ADD COLUMN IF NOT EXISTS parent_job_id UUID;
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent ON gmaps_jobs(parent_job_id);

-- Add task tracking columns to jobs_queue
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS total_tasks INTEGER DEFAULT 0;
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS completed_tasks INTEGER DEFAULT 0;

-- Index for efficient task count queries
CREATE INDEX IF NOT EXISTS idx_gmaps_jobs_parent_status ON gmaps_jobs(parent_job_id, status);

COMMIT;
