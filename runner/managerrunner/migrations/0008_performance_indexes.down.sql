-- Down migration for 0008

DROP INDEX IF EXISTS idx_business_emails_listing_email;
DROP INDEX IF EXISTS idx_emails_acceptable_email;
DROP INDEX IF EXISTS idx_business_listings_job_id_id;
DROP INDEX IF EXISTS idx_business_listings_created_at_id;
