-- Migration 0002: Add Proxy Sources Table

CREATE TABLE IF NOT EXISTS proxy_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_proxy_sources_created_at ON proxy_sources(created_at DESC);
