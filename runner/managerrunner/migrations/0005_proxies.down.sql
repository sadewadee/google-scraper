-- Migration 0005: Proxies Table (DOWN)

BEGIN;

DROP INDEX IF EXISTS idx_proxies_last_checked;
DROP INDEX IF EXISTS idx_proxies_country;
DROP INDEX IF EXISTS idx_proxies_protocol;
DROP INDEX IF EXISTS idx_proxies_status;
DROP TABLE IF EXISTS proxies;

COMMIT;
