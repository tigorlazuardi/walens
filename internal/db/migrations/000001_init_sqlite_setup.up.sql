-- +goose up
-- +goose no transaction
-- Migration 1: SQLite baseline setup only.
-- Keep connection-scoped pragmas in db.Open(); apply persistent setup here.

PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
