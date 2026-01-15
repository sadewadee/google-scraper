BEGIN;

-- Rollback Dashboard → DSN Bridge Migration

-- Remove indexes first
DROP INDEX IF EXISTS idx_gmaps_jobs_parent_status;
DROP INDEX IF EXISTS idx_gmaps_jobs_parent;

-- Remove columns
ALTER TABLE gmaps_jobs DROP COLUMN IF EXISTS parent_job_id;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS total_tasks;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS completed_tasks;

COMMIT;
