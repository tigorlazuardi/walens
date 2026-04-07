-- +goose up
-- Migration 3: Create business schema tables.
-- Tables: devices, sources, source_schedules, device_source_subscriptions,
-- images, tags, image_tags, image_assignments, image_locations,
-- image_thumbnails, image_blacklists, jobs.

-- devices table: user-managed wallpaper consumer devices.
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    screen_width INTEGER NOT NULL,
    screen_height INTEGER NOT NULL,
    min_image_width INTEGER NOT NULL DEFAULT 0,
    max_image_width INTEGER NOT NULL DEFAULT 0,
    min_image_height INTEGER NOT NULL DEFAULT 0,
    max_image_height INTEGER NOT NULL DEFAULT 0,
    min_filesize INTEGER NOT NULL DEFAULT 0,
    max_filesize INTEGER NOT NULL DEFAULT 0,
    is_adult_allowed INTEGER NOT NULL DEFAULT 0,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    aspect_ratio_tolerance REAL NOT NULL DEFAULT 0.15,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_slug ON devices(slug);

-- sources table: user-created configured source rows.
CREATE TABLE IF NOT EXISTS sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    params TEXT NOT NULL DEFAULT '{}',
    lookup_count INTEGER NOT NULL DEFAULT 0,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sources_name ON sources(name);

-- source_schedules table: cron schedules for source rows.
CREATE TABLE IF NOT EXISTS source_schedules (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_source_schedules_source_id ON source_schedules(source_id);

-- device_source_subscriptions table: connects devices to source rows.
CREATE TABLE IF NOT EXISTS device_source_subscriptions (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    source_id TEXT NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_source_subscriptions_device_source ON device_source_subscriptions(device_id, source_id);

-- images table: canonical downloaded image identity records.
CREATE TABLE IF NOT EXISTS images (
    id TEXT PRIMARY KEY,
    source_id TEXT,
    unique_identifier TEXT NOT NULL,
    source_type TEXT NOT NULL,
    original_filename TEXT,
    preview_url TEXT,
    origin_url TEXT,
    source_item_identifier TEXT,
    original_identifier TEXT,
    uploader TEXT,
    artist TEXT,
    mime_type TEXT,
    file_size_bytes INTEGER,
    width INTEGER,
    height INTEGER,
    aspect_ratio REAL,
    is_adult INTEGER NOT NULL DEFAULT 0,
    is_favorite INTEGER NOT NULL DEFAULT 0,
    json_meta TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_images_source_id ON images(source_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_images_unique_identifier ON images(source_id, unique_identifier);

-- tags table: normalized tag names.
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_normalized_name ON tags(normalized_name);

-- image_tags table: join between images and tags.
CREATE TABLE IF NOT EXISTS image_tags (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_tags_image_tag ON image_tags(image_id, tag_id);

-- image_assignments table: tracks which devices an image is assigned to.
CREATE TABLE IF NOT EXISTS image_assignments (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_assignments_image_device ON image_assignments(image_id, device_id);

-- image_locations table: tracks concrete file paths on disk for images.
CREATE TABLE IF NOT EXISTS image_locations (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    path TEXT NOT NULL,
    storage_kind TEXT NOT NULL,
    is_primary INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_locations_path ON image_locations(path);
CREATE INDEX IF NOT EXISTS idx_image_locations_image_id ON image_locations(image_id);
CREATE INDEX IF NOT EXISTS idx_image_locations_device_id ON image_locations(device_id);

-- image_thumbnails table: generated thumbnail files derived from images.
CREATE TABLE IF NOT EXISTS image_thumbnails (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL,
    path TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size_bytes INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_thumbnails_image_id ON image_thumbnails(image_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_image_thumbnails_path ON image_thumbnails(path);

-- image_blacklists table: blocks download of unwanted images by source+identifier.
CREATE TABLE IF NOT EXISTS image_blacklists (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    unique_identifier TEXT NOT NULL,
    reason TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_blacklists_source_identifier ON image_blacklists(source_id, unique_identifier);

-- jobs table: persisted execution records for source sync/download jobs.
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    job_type TEXT NOT NULL,
    source_id TEXT,
    source_name TEXT,
    source_type TEXT,
    status TEXT NOT NULL,
    trigger_kind TEXT NOT NULL,
    run_after INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    duration_ms INTEGER,
    requested_image_count INTEGER NOT NULL DEFAULT 0,
    downloaded_image_count INTEGER NOT NULL DEFAULT 0,
    reused_image_count INTEGER NOT NULL DEFAULT 0,
    hardlinked_image_count INTEGER NOT NULL DEFAULT 0,
    copied_image_count INTEGER NOT NULL DEFAULT 0,
    stored_image_count INTEGER NOT NULL DEFAULT 0,
    skipped_image_count INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    error_message TEXT,
    json_input TEXT NOT NULL DEFAULT '{}',
    json_result TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_source_id ON jobs(source_id);
CREATE INDEX IF NOT EXISTS idx_jobs_run_after ON jobs(run_after);

-- +goose down
-- Reverse order of drops to respect foreign key constraints.
DROP INDEX IF EXISTS idx_jobs_run_after;
DROP INDEX IF EXISTS idx_jobs_source_id;
DROP INDEX IF EXISTS idx_jobs_status;
DROP TABLE IF EXISTS jobs;

DROP INDEX IF EXISTS idx_image_blacklists_source_identifier;
DROP TABLE IF EXISTS image_blacklists;

DROP INDEX IF EXISTS idx_image_thumbnails_path;
DROP INDEX IF EXISTS idx_image_thumbnails_image_id;
DROP TABLE IF EXISTS image_thumbnails;

DROP INDEX IF EXISTS idx_image_locations_device_id;
DROP INDEX IF EXISTS idx_image_locations_image_id;
DROP INDEX IF EXISTS idx_image_locations_path;
DROP TABLE IF EXISTS image_locations;

DROP INDEX IF EXISTS idx_image_assignments_image_device;
DROP TABLE IF EXISTS image_assignments;

DROP INDEX IF EXISTS idx_image_tags_image_tag;
DROP TABLE IF EXISTS image_tags;

DROP INDEX IF EXISTS idx_tags_normalized_name;
DROP TABLE IF EXISTS tags;

DROP INDEX IF EXISTS idx_images_unique_identifier;
DROP INDEX IF EXISTS idx_images_source_id;
DROP TABLE IF EXISTS images;

DROP INDEX IF EXISTS idx_device_source_subscriptions_device_source;
DROP TABLE IF EXISTS device_source_subscriptions;

DROP INDEX IF EXISTS idx_source_schedules_source_id;
DROP TABLE IF EXISTS source_schedules;

DROP INDEX IF EXISTS idx_sources_name;
DROP TABLE IF EXISTS sources;

DROP INDEX IF EXISTS idx_devices_slug;
DROP TABLE IF EXISTS devices;
