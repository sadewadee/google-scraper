-- Migration 0003: Add location search and grid coverage support
-- SQLite version for Dashboard/Web UI

-- Add new columns to jobs_queue table
ALTER TABLE jobs_queue ADD COLUMN location_name TEXT;
ALTER TABLE jobs_queue ADD COLUMN bounding_box TEXT; -- JSON as TEXT
ALTER TABLE jobs_queue ADD COLUMN coverage_mode TEXT DEFAULT 'single';
ALTER TABLE jobs_queue ADD COLUMN grid_size INTEGER DEFAULT 3;
