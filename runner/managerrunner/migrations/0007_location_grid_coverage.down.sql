-- Rollback migration 0007: Remove location search and grid coverage support

BEGIN;

ALTER TABLE jobs_queue DROP CONSTRAINT IF EXISTS valid_coverage_mode;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS grid_points;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS coverage_mode;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS boundingbox;
ALTER TABLE jobs_queue DROP COLUMN IF EXISTS location_name;

COMMIT;
