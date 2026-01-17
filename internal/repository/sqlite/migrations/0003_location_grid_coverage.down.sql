-- Migration 0003: Rollback location search and grid coverage support
-- Note: SQLite 3.35.0+ supports DROP COLUMN. For older versions, table recreation is needed.

-- Remove columns (requires SQLite 3.35.0+)
ALTER TABLE jobs_queue DROP COLUMN grid_points;
ALTER TABLE jobs_queue DROP COLUMN coverage_mode;
ALTER TABLE jobs_queue DROP COLUMN boundingbox;
ALTER TABLE jobs_queue DROP COLUMN location_name;
