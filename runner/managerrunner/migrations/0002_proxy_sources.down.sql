-- Migration 0002: Proxy Sources (Rollback)
-- Drops the proxy_sources table

BEGIN;

DROP TABLE IF EXISTS proxy_sources;

COMMIT;
