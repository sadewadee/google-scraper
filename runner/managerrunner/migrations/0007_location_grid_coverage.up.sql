-- Migration 0007: Add location search and grid coverage support
-- Allows users to search by city/region and cover the entire area with grid points

BEGIN;

-- Add new columns to jobs_queue table
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS location_name TEXT;
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS boundingbox JSONB;
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS coverage_mode TEXT DEFAULT 'single';
ALTER TABLE jobs_queue ADD COLUMN IF NOT EXISTS grid_points INT DEFAULT 1;

-- Add constraint for coverage_mode
ALTER TABLE jobs_queue DROP CONSTRAINT IF EXISTS valid_coverage_mode;
ALTER TABLE jobs_queue ADD CONSTRAINT valid_coverage_mode
    CHECK (coverage_mode IN ('single', 'full'));

-- Add comment for documentation
COMMENT ON COLUMN jobs_queue.location_name IS 'Human-readable location name from geocoding (e.g., "Bandung, West Java, Indonesia")';
COMMENT ON COLUMN jobs_queue.boundingbox IS 'Bounding box from geocoding: {"min_lat": float, "max_lat": float, "min_lon": float, "max_lon": float}';
COMMENT ON COLUMN jobs_queue.coverage_mode IS 'single = search from center point only, full = generate grid of search points based on radius';
COMMENT ON COLUMN jobs_queue.grid_points IS 'Number of grid points generated for full coverage mode';

COMMIT;
