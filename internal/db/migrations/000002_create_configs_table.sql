-- +goose up
-- Migration 2: Create configs table for application configuration.
-- This table stores the full serialized application config as JSON.

CREATE TABLE IF NOT EXISTS configs (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    value TEXT NOT NULL DEFAULT '{}',
    updated_at INTEGER NOT NULL
);

INSERT OR IGNORE INTO configs (id, value, updated_at) VALUES (1, '{}', 0);

-- +goose down
DROP TABLE IF EXISTS configs;
