-- Migration 0002: Proxy Sources
-- Creates table for persisting proxy sources

BEGIN;

CREATE TABLE IF NOT EXISTS proxy_sources (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
