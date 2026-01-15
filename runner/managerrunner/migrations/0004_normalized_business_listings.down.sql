-- Migration 0004 DOWN: Remove normalized schema
-- (Embedded migration for managerrunner auto-migrate)

BEGIN;

-- Drop views first (depend on tables)
DROP VIEW IF EXISTS v_email_validation_queue;
DROP VIEW IF EXISTS v_business_listings_with_emails;

-- Drop functions
DROP FUNCTION IF EXISTS backfill_normalized_listings(INTEGER);
DROP FUNCTION IF EXISTS mark_email_validation_error(TEXT, TEXT);
DROP FUNCTION IF EXISTS update_email_validation(TEXT, TEXT, NUMERIC, BOOLEAN, BOOLEAN, BOOLEAN, BOOLEAN, BOOLEAN, TEXT);

-- Drop trigger
DROP TRIGGER IF EXISTS trg_populate_normalized_listings ON results;
DROP FUNCTION IF EXISTS populate_normalized_listings();

-- Drop tables (in order of dependencies)
DROP TABLE IF EXISTS business_emails;
DROP TABLE IF EXISTS emails;
DROP TABLE IF EXISTS business_listings;

COMMIT;
