-- Migration 0005: Proxies Table
-- Stores fetched proxies with status and metrics

BEGIN;

CREATE TABLE IF NOT EXISTS proxies (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL,           -- IPv4 or IPv6
    port INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL DEFAULT 'socks5',  -- socks5, socks4, http, https
    country VARCHAR(5),                 -- Country code (US, SG, etc)

    -- Metrics from source
    uptime DECIMAL(5,2),               -- Uptime percentage (0-100)
    response_time DECIMAL(6,3),        -- Response time in seconds

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, healthy, dead, banned
    last_checked TIMESTAMPTZ,
    last_used TIMESTAMPTZ,
    fail_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,

    -- Source tracking
    source_id BIGINT REFERENCES proxy_sources(id) ON DELETE SET NULL,
    source_url TEXT,                    -- Original source URL (for sources without DB entry)

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint on IP:port
    UNIQUE(ip, port)
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_protocol ON proxies(protocol);
CREATE INDEX IF NOT EXISTS idx_proxies_country ON proxies(country);
CREATE INDEX IF NOT EXISTS idx_proxies_last_checked ON proxies(last_checked);

COMMIT;
