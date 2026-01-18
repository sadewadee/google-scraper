-- Migration 0008: Performance indexes to avoid 504 timeouts on list pages
-- These indexes optimize the JOIN patterns used in business listing queries

BEGIN;

-- Composite index for business_emails join pattern
-- This helps the query: JOIN business_emails be ON be.business_listing_id = bl.id
CREATE INDEX IF NOT EXISTS idx_business_emails_listing_email
    ON business_emails(business_listing_id, email_id);

-- Covering index for emails table used in aggregation
-- Includes columns used in SELECT to enable index-only scans
CREATE INDEX IF NOT EXISTS idx_emails_acceptable_email
    ON emails(id, email, validation_status, api_score, is_acceptable)
    WHERE is_acceptable IS NOT NULL;

-- Optimize COUNT queries by job_id
CREATE INDEX IF NOT EXISTS idx_business_listings_job_id_id
    ON business_listings(job_id, id);

-- Optimize the default sort (created_at DESC) with covering columns
CREATE INDEX IF NOT EXISTS idx_business_listings_created_at_id
    ON business_listings(created_at DESC, id);

-- Update table statistics for better query planning
ANALYZE business_listings;
ANALYZE business_emails;
ANALYZE emails;

COMMIT;
