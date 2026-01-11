BEGIN;

-- Drop triggers
DROP TRIGGER IF EXISTS update_jobs_queue_updated_at ON jobs_queue;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_results_job_id;
DROP INDEX IF EXISTS idx_workers_heartbeat;
DROP INDEX IF EXISTS idx_jobs_queue_pending;
DROP INDEX IF EXISTS idx_jobs_queue_worker_id;
DROP INDEX IF EXISTS idx_jobs_queue_created_at;
DROP INDEX IF EXISTS idx_jobs_queue_status;

-- Remove job_id from results
ALTER TABLE results DROP COLUMN IF EXISTS job_id;

-- Drop tables
DROP TABLE IF EXISTS workers;
DROP TABLE IF EXISTS jobs_queue;

COMMIT;
