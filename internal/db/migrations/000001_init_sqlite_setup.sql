-- +goose no transaction
-- +goose up
-- Migration 1: SQLite baseline setup only.
-- Keep connection-scoped pragmas in db.Open(); apply persistent setup here.

PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

-- +goose down
PRAGMA journal_mode = DELETE;
PRAGMA synchronous = FULL;
