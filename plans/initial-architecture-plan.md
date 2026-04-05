# Walens Initial Architecture Plan

## Goals

- Build Walens as a wallpaper collection app server with very high ease of deployment.
- Support direct binary deployment on `Windows`, `Linux`, and `macOS`.
- Support both `arm64` and `x86_64` targets.
- Support one Docker image deployment for server runtime.
- Keep the runtime model strictly single-process, with no high-availability assumptions.
- Keep backend dependencies pure Go and avoid CGO, native bindings, and FFI.
- Keep the frontend stack free from native runtime bindings in local and CI usage.

## Deployment-First Principles

Walens should be designed around operational simplicity first.

### Non-negotiable runtime constraints

1. One process runs the whole application.
2. HTTP server, scheduler, queue, and job runner live in that same process.
3. No HA coordination, distributed lock, leader election, or external queue.
4. No CGO.
5. No native bindings.
6. No FFI-based runtime dependencies.
7. Database is local SQLite accessed through a pure-Go driver.

### Target deliverables

- One statically linked binary per OS/arch target.
- One Docker image for container deployment.
- One writable directory or volume for SQLite database and optional app data.

### Supported targets

- `windows/amd64`
- `windows/arm64`
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

## Product Scope for Current Phase

### Core entities

1. **Device**
   - User-managed wallpaper consumer.
   - Stores screen dimensions.
   - Stores adult content policy.
   - Stores allowed aspect ratio tolerance.

2. **Source Definition**
   - Persisted in database.
   - Represents one user-created source row in `sources`.
   - Each row points to one source implementation unique name from codebase and stores params for that configured row.
   - Example: source implementation `booru`, configured source row `anime-pics-main`, params `{ "tags": ["landscape"] }`.

3. **Source Schedule**
   - Persisted in database.
   - One source can have many cron schedules.
   - Used by background scheduler to enqueue sync jobs.

4. **Device Source Subscription**
   - Persisted in database.
   - Connects a device to a row in `sources`.
   - This is what makes the device a candidate when that source row runs on schedule.

5. **Image**
   - Persisted downloaded image identity record.
   - Uses source-provided `unique_identifier` for best-effort dedupe.
   - Represents canonical local image content when a file has already been downloaded.

6. **Image Assignment**
   - Persisted relation between an image and a device.
   - Used to track whether an image is already assigned to a given device.

7. **Image Location**
   - Persisted tracker for every file path on disk that points to an image.
   - Supports hard-link tracking and cleanup.

8. **Image Blacklist**
   - Persisted deny-list keyed by source image unique identifier.
   - Prevents future download/redownload of blacklisted images.

9. **Job**
   - Persisted execution record for source sync/download jobs.
   - Created for a `sources` row, not directly for a source implementation type.
   - Stores requested work, runtime status, timings, counts, and errors.

### Clarified source model

- Source implementations are defined in Go code and registered in a runtime registry.
- Each implementation has a unique code-level name, for example `booru` or `reddit`.
- The database `sources` table stores user-created configured source rows.
- Each source row references one implementation unique name and has its own params.
- Each source row can be enabled/disabled and can have many schedules.
- Devices subscribe to source rows from the `sources` table, not directly to implementation types.
- When a source row schedule runs, subscribed devices for that row become the candidate set for image fetching and assignment.

### Phase 1 behavior

- CRUD devices.
- CRUD sources.
- CRUD source schedules.
- CRUD device source subscriptions.
- Register source implementations in code and expose them by API.
- Allow source rows to reference a registered source implementation name plus params.
- Run scheduler in-process and enqueue jobs based on `sources` row schedules.
- Run queue and job runner in-process.
- Persist jobs in database.
- Requeue unfinished jobs on boot.
- Download and persist images from source-row jobs into local SQLite database.
- Perform best-effort dedupe on downloaded images using source-provided `unique_identifier`.
- Reuse existing local image files across devices without redownload when possible.
- Prefer hard link creation for additional device-visible paths, then fallback to file copy.
- Track all image file locations on disk for later cleanup.
- Query images for a specific device using device filters and subscribed sources.
- Support optional Basic Auth for self-hosted protection, configurable from env or config file.

### Product posture

Walens should stay intentionally small in scope.

- not a user platform
- not a collaborative system
- not a content management product
- not a workflow-heavy business app

Core purpose only:

- run wallpaper collection jobs on schedule
- filter images that match device constraints
- download and store images efficiently
- expose a small self-hosted web UI for managing that flow

### Explicit non-goals for now

- High availability.
- Multi-node scheduling.
- External queue broker.
- Distributed workers.
- User accounts, user management, and web-editable credentials.
- roles, permissions, sessions, profiles, invitations, or any account lifecycle features.
- Multi-tenant permissions.
- Advanced recommendation/ranking.
- Complex local binary asset pipeline beyond what is required for initial sync/download behavior.

## High-Level Architecture

### Backend

- Language: Go.
- Runtime: standard `net/http` server.
- API framework: Huma.
- Database: SQLite via pure-Go driver.
- Query layer: Go-Jet generated models + query builder.
- Migration tool: golang-migrate.
- Frontend dev integration: `github.com/olivere/vite`.
- Background runtime: in-process scheduler + in-memory queue + in-process job runner.

### Frontend

- SvelteKit configured as SPA / full client-side app.
- TanStack Svelte Query.
- `shadcn-svelte`.
- `openapi-fetch` for typed API integration.

### Runtime topology

Single process only:

- one HTTP server
- one scheduler manager
- one in-memory queue
- one job runner supervisor
- one SQLite connection pool / database handle

No separate worker binary should be required.

### Access control model

Walens should support optional simple access protection suitable for self-hosted deployments.

Rules:

- Basic Auth support is optional and can be disabled entirely.
- Auth is configured only through config file or env vars.
- Username/password cannot be changed from the web UI.
- When auth is enabled, request validation should:
  1. first check `Authorization` header
  2. if absent, check HTTP-only auth cookie
  3. reject with `401` if credentials are invalid or missing
- Browser users without valid cookie should be able to reach a simple login page.
- Future mobile or external clients are expected to use the `Authorization` header.

Security intent:

- not state-of-the-art identity or session management
- good enough to reduce casual brute force, bot crawling, and unwanted access to potentially adult content
- should be understood as deployment protection only, not application-level user management

## Dependency Constraints

### Backend dependency policy

All backend libraries must satisfy:

- pure Go implementation
- no CGO
- no native shared libraries
- no FFI bridge

Key implications:

- SQLite driver must be `modernc.org/sqlite` or equivalent pure-Go option.
- Image processing libraries with native dependencies should be avoided unless strictly optional and isolated.
- Scheduler, queue, and cron libraries must be Go-only.

### Frontend dependency policy

Frontend build and dev tools should avoid native bindings where reasonably possible.

Practical rule for the project:

- do not introduce dependencies that require platform-native compilation or runtime native modules for normal local development, CI, or production asset builds.

This keeps cross-platform development closer to the same operational simplicity goal as the backend.

## Proposed Project Layout

```text
walens/
  cmd/
    walens/
      main.go
  internal/
    app/
    config/
    db/
      migrations/
      generated/
      queries/
    routes/
      configs/
      register.go
      devices/
      images/
      jobs/
      source_schedules/
      source_types/
      sources/
    services/
      configs/
      devices/
      images/
      jobs/
      source_schedules/
      source_types/
      sources/
    scheduler/
    queue/
    runner/
    ingest/
    http/
      middleware/
    frontend/
  frontend/
    src/
    static/
    package.json
    vite.config.ts
    svelte.config.js
  plans/
```

## Data Model Plan

### 1. `devices`

Suggested columns:

- `id` TEXT primary key
- `name` TEXT not null
- `slug` TEXT not null
- `screen_width` INTEGER not null
- `screen_height` INTEGER not null
- `min_image_width` INTEGER not null default 0
- `max_image_width` INTEGER not null default 0
- `min_image_height` INTEGER not null default 0
- `max_image_height` INTEGER not null default 0
- `min_filesize` INTEGER not null default 0
- `max_filesize` INTEGER not null default 0
- `is_adult_allowed` INTEGER not null default 0
- `aspect_ratio_tolerance` REAL not null default 0.15
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Notes:

- `aspect_ratio_tolerance` should be absolute ratio delta tolerance.
- Example: device ratio `16/9 ~= 1.777`, wallpaper ratio accepted if `abs(wallpaper_ratio - device_ratio) <= tolerance`.
- `min_image_width`, `max_image_width`, `min_image_height`, and `max_image_height` provide explicit dimension filters in addition to screen-based matching.
- `min_filesize` and `max_filesize` provide explicit file size filters for matched images.
- a zero value for these bounds should mean "not set" / no extra limit unless a stricter rule is later chosen.
- timestamp fields such as `created_at` and `updated_at` should use INTEGER Unix milliseconds.

### 2. `sources`

This is now a first-class table.

Suggested columns:

- `id` TEXT primary key
- `name` TEXT not null
- `source_type` TEXT not null
- `params` TEXT not null default '{}'
- `lookup_count` INTEGER not null default 0
- `is_enabled` INTEGER not null default 1
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Semantics:

- `name` is the user-defined unique name of the source row in `sources`.
- `source_type` is the code-registered source implementation name, for example `booru` or `reddit`.
- `params` stores stringified JSON for that specific source row.
- `lookup_count` controls how many upstream source items/posts should be inspected per run.
- if `lookup_count = 0`, Walens should use the default from the source implementation.
- `lookup_count` is a lookup budget, not a guaranteed image result count.
- non-image posts, skipped posts, and deduped results still count toward `lookup_count`.
- this is important so users can control how much upstream data is read from a source and reduce the chance of hitting that source's rate limits.
- example meaning: "check the latest 300 posts", not "return 300 downloadable images".
- Devices subscribe to this source row by `source_id`.
- Schedules also belong to this source row by `source_id`.

Constraints:

- unique index on `name`
- validate `source_type` against runtime registry before insert/update

### 3. `source_schedules`

One source can have many schedules.

Suggested columns:

- `id` TEXT primary key
- `source_id` TEXT not null references `sources(id)` on delete cascade
- `cron_expr` TEXT not null
- `is_enabled` INTEGER not null default 1
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Notes:

- Cron format should be standardized early, preferably 5-field cron unless there is a strong reason to support seconds.
- Validation should happen at API boundary and in domain service.

### 4. `device_source_subscriptions`

Suggested columns:

- `id` TEXT primary key
- `device_id` TEXT not null references `devices(id)` on delete cascade
- `source_id` TEXT not null references `sources(id)` on delete cascade
- `is_enabled` INTEGER not null default 1
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Constraints:

- unique index on `(device_id, source_id)`

### 5. `images`

This table stores canonical downloaded image identity records.

Suggested columns:

- `id` TEXT primary key
- `source_id` TEXT references `sources(id)`
- `unique_identifier` TEXT not null
- `source_type` TEXT not null
- `original_filename` TEXT
- `preview_url` TEXT
- `origin_url` TEXT
- `source_item_identifier` TEXT
- `original_identifier` TEXT
- `uploader` TEXT
- `artist` TEXT
- `mime_type` TEXT
- `file_size_bytes` INTEGER
- `width` INTEGER
- `height` INTEGER
- `aspect_ratio` REAL
- `is_adult` INTEGER not null default 0
- `is_favorite` INTEGER not null default 0
- `json_meta` TEXT not null default '{}'
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Semantics:

- `unique_identifier` is generated by the source implementation.
- `unique_identifier` is best effort only.
- duplicates can still exist in reality if the source emits unstable IDs or different sources describe the same file differently.
- `original_identifier` should be used for external/source-owned identifiers rather than `original_id`.
- `source_item_identifier` is the external source item/post identifier, not an internal Walens ID.

Constraints:

- unique index on `(source_id, unique_identifier)` is recommended for phase 1.

Tag note:

- image/tag metadata should use dedicated relational tables for better dedupe and filtering

### 6. `tags`

This table stores unique tags in normalized relational form.

Suggested columns:

- `id` TEXT primary key
- `name` TEXT not null
- `normalized_name` TEXT not null
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Constraints:

- unique index on `normalized_name`

Semantics:

- `normalized_name` should be a case-insensitive canonical form, for example lowercase trimmed text
- tag uniqueness should be enforced case-insensitively through `normalized_name`

### 7. `image_tags`

Join table between images and tags.

Suggested columns:

- `id` TEXT primary key
- `image_id` TEXT not null references `images(id)` on delete cascade
- `tag_id` TEXT not null references `tags(id)` on delete cascade
- `created_at` INTEGER not null

Constraints:

- unique index on `(image_id, tag_id)`

### 8. `image_assignments`

This table tracks which devices an image has been assigned to.

Suggested columns:

- `id` TEXT primary key
- `image_id` TEXT not null references `images(id)` on delete cascade
- `device_id` TEXT not null references `devices(id)` on delete cascade
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Constraints:

- unique index on `(image_id, device_id)`

Semantics:

- assignment is device-specific state
- assignment may exist even when the file is temporarily missing on disk
- assignment state is used by the runner to decide skip, redownload, or materialize behavior

### 9. `image_locations`

This table tracks every concrete file path on disk for an image.

Suggested columns:

- `id` TEXT primary key
- `image_id` TEXT not null references `images(id)` on delete cascade
- `device_id` TEXT not null references `devices(id)` on delete cascade
- `path` TEXT not null
- `storage_kind` TEXT not null
- `is_primary` INTEGER not null default 0
- `is_active` INTEGER not null default 1
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Suggested enum-like values:

- `storage_kind`: `canonical`, `hardlink`, `copy`

Semantics:

- `canonical` is the first device-specific stored file path for the image.
- `hardlink` is an additional filesystem path pointing to the same content through a hard link.
- `copy` is fallback storage when hard links cannot be created.
- image locations are always tracked per device because filesystem placement is device-oriented by design.
- one image can appear in many device-specific paths.

Constraints:

- unique index on `path`

### 10. `image_thumbnails`

This table tracks generated thumbnail files derived from downloaded images.

Suggested columns:

- `id` TEXT primary key
- `image_id` TEXT not null references `images(id)` on delete cascade
- `path` TEXT not null
- `width` INTEGER not null
- `height` INTEGER not null
- `file_size_bytes` INTEGER
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Constraints:

- unique index on `image_id`
- unique index on `path`

Semantics:

- thumbnail generation is a post-download step
- thumbnails are optimized for UI listing and preview, not for device assignment
- deleting an image should also delete its thumbnail file if present

### 11. `image_blacklists`

This table blocks download of known unwanted images.

Suggested columns:

- `id` TEXT primary key
- `source_id` TEXT references `sources(id)` on delete cascade
- `unique_identifier` TEXT not null
- `reason` TEXT
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Constraints:

- unique index on `(source_id, unique_identifier)`

Semantics:

- blacklist is checked before any download begins
- if a unique identifier is blacklisted, that image must never be downloaded or redownloaded for that source

### 12. `jobs`

Jobs are first-class persisted records.

Suggested columns:

- `id` TEXT primary key
- `job_type` TEXT not null
- `source_id` TEXT references `sources(id)`
- `source_name` TEXT
- `source_type` TEXT
- `status` TEXT not null
- `trigger_kind` TEXT not null
- `run_after` INTEGER not null
- `started_at` INTEGER
- `finished_at` INTEGER
- `duration_ms` INTEGER
- `requested_image_count` INTEGER not null default 0
- `downloaded_image_count` INTEGER not null default 0
- `reused_image_count` INTEGER not null default 0
- `hardlinked_image_count` INTEGER not null default 0
- `copied_image_count` INTEGER not null default 0
- `stored_image_count` INTEGER not null default 0
- `skipped_image_count` INTEGER not null default 0
- `message` TEXT
- `error_message` TEXT
- `json_input` TEXT not null default '{}'
- `json_result` TEXT not null default '{}'
- `created_at` INTEGER not null
- `updated_at` INTEGER not null

Suggested enum-like values:

- `job_type`: `source_sync`, `source_download`
- `status`: `queued`, `running`, `succeeded`, `failed`, `cancelled`
- `trigger_kind`: `manual`, `schedule`, `recovery`

Notes:

- `requested_image_count` captures how many images should be fetched/downloaded.
- `downloaded_image_count` captures how many images were actually retrieved.
- `reused_image_count` captures how many images were satisfied from existing local content.
- `hardlinked_image_count` captures how many output paths were created as hard links.
- `copied_image_count` captures how many output paths required copy fallback.
- `stored_image_count` captures how many image records were persisted or updated.
- `skipped_image_count` captures how many candidate items were skipped during processing.
- `message` stores non-error informational result text for cases such as "did not run because no enabled devices subscribe to this source".
- `duration_ms` stores job duration in integer milliseconds using the duration wrapper convention.
- `error_message` stores summarized terminal failure reason.
- `json_input` stores source params / execution request snapshot.
- `json_result` stores extensible metadata such as source cursor, warnings, or counts by category.

### 13. `job_attempts` (optional)

Do not create this in phase 1 unless retry complexity grows.

Reasoning:

- One `jobs` table is enough at first.
- Retry and attempt history can be added later if needed.

## Source System Plan

### Interface design

Each source implementation should define:

- stable source implementation name, for example `booru`, `reddit`
- human-readable label
- parameter validation
- parameter schema as `*huma.Schema` for frontend form generation
- default lookup count when a source row sets `lookup_count = 0`
- fetch logic that yields image metadata lazily through iterator
- best-effort `unique_identifier` generation for downloadable image items

Suggested conceptual interface:

```go
type Source interface {
    TypeName() string
    DisplayName() string
    ValidateParams(raw json.RawMessage) error
    ParamSchema() *huma.Schema
    DefaultLookupCount() int
    Fetch(ctx context.Context, req FetchRequest) iter.Seq2[ImageMetadata, error]
    BuildUniqueID(item ImageMetadata) (string, error)
}

type FetchRequest struct {
    Params      []byte
    LookupCount int
}
```

Interface notes:

- `Fetch` should stream `ImageMetadata` items lazily rather than returning one large in-memory slice
- iterator-based fetching allows downstream download work to start before the full source scan completes
- `Params` can be stored in the app as stringified JSON, but the source-facing request can use `[]byte` for direct JSON handling

### Registry pattern

- Build a `source.Registry` in Go.
- Register implementations on startup.
- API exposes available source implementation types from registry.
- `sources.source_type` must match a registry entry.
- `sources.params` must pass that implementation's validation.

### Source row vs source implementation

Example:

- runtime implementation type: `booru`
- configured database source row in `sources`: `anime-pics-main`
- subscribed devices: `living-room-tv`, `office-monitor`

Execution meaning:

- scheduler fires for `anime-pics-main`
- a job is created for that `sources.id`
- the runner calls `Fetch(...)` on the `booru` implementation with params from `anime-pics-main`
- devices subscribed to `anime-pics-main` are the candidate set for downstream image fetching and assignment decisions

Download/storage meaning:

- if the source implementation returns an item whose `unique_identifier` already exists locally, Walens should reuse the existing file when possible
- if another device needs the same image path materialized, Walens should create a hard link first
- if hard link creation fails, Walens should fall back to file copy

This separation is important because many configured source rows can use the same implementation type with different params and schedules.

## Scheduling, Queue, and Job Runner Plan

This is now part of the core architecture, not a later add-on.

### Scheduler responsibilities

- Load all enabled sources and their enabled schedules.
- Register cron triggers in memory.
- Enqueue jobs into the in-memory queue when a source-row cron fires.
- Rebuild scheduler state whenever sources or source schedules change.
- Validate schedule proximity and emit warnings when schedule times are too close.

### Scheduler resync behavior

Any create/update/delete that changes:

- `sources`
- `source_schedules`
- enabled/disabled state

should trigger scheduler resync.

Recommended implementation:

- API/domain mutation calls a `scheduler.Reload()` method after successful transaction commit.
- `Reload()` rebuilds active cron entries from DB snapshot.

### Schedule proximity warning rule

For the same configured source unique `name`, the scheduler should warn if schedules are too close.

Requested warning rule:

- if a day-of-week and time-of-day occurrence for the same source name is less than 5 minutes apart, emit warning

Practical interpretation:

- analyze the next occurrences generated by each enabled cron entry for the same source
- if any pair overlaps within `< 5m`, record a warning

Recommended output shape:

- return warnings in source schedule validation API responses
- persist latest warning snapshot in memory or recompute on demand
- optionally store a warning text on the source row later if UI needs cheap display

Important note:

- Cron overlap analysis can get tricky across complex expressions.
- For phase 1, a bounded lookahead strategy is good enough, for example checking occurrences across the next 14 days.

### Queue responsibilities

- In-memory only.
- Hold pending job IDs.
- Support enqueue, dequeue, visibility of queue size, and graceful shutdown draining behavior.

Recommended simplicity:

- queue payload is job ID only
- authoritative job state stays in database

### Boot recovery behavior

At app startup:

1. load jobs with status `queued` or `running`
2. convert `running` to `queued` or mark recovery metadata
3. enqueue them back into memory queue
4. start normal scheduling after recovery pass

Rationale:

- the process model is single-process and crash recovery only needs local DB state restoration

### Job runner responsibilities

- Pull job IDs from queue.
- Lock/update job status to `running`.
- Resolve source configuration from DB.
- Invoke source implementation fetch iterator.
- Persist results, counts, errors, and timings.
- Resolve or create `images` rows using source-provided `unique_identifier`.
- Reuse existing local image content without redownload when possible.
- Generate thumbnail file after successful canonical image download or reuse when thumbnail is missing.
- Materialize target file paths using hard links first, then copy fallback.
- Persist device assignment state into `image_assignments`.
- Persist all tracked file paths into `image_locations`.
- Persist thumbnail metadata/path into `image_thumbnails`.
- Mark terminal status.

### Background fetch/run flow

The background source runner should follow this behavior:

1. check whether the source row is enabled
2. if disabled, exit immediately and do not create any job log
3. when cron triggers for an enabled source, load enabled subscribed device candidates
4. if no enabled subscribed devices exist, create a job log with an English message like `did not run because no enabled devices subscribe to this source`
5. start fetching image metadata from the source
6. compare each fetched image metadata item against the candidate devices
7. if no devices match, skip that item and continue
8. if at least one device matches, send the image metadata plus eligible device list into downstream processing

### Assignment/materialization rules

For each eligible `(image, device)` combination, the worker should apply these rules:

1. if the image is already assigned to that device and the file exists on disk, skip
2. if the image is already assigned to that device but the file is missing on disk, download/materialize it again for that device
3. if the image is assigned to another device but not this device, create hard link if possible, otherwise copy
4. if the image is assigned to another device but the source file is missing, download for the current device and assign only to the current device

Important rule:

- do not implicitly reassign other devices during recovery of one missing file path

### Blacklist behavior

- before download starts, check `image_blacklists` using the relevant `(source_id, unique_identifier)`
- if blacklisted, never download or redownload that image
- blacklisted items may still count against source lookup budget, but must not proceed into download

### Download behavior

- download into a temporary location first
- only after successful completion should the file be moved or copied into the first device image location
- thumbnail generation happens after the canonical device file is available

Fetch/worker note:

- iterator output from `Fetch(...)` should be usable to feed downstream download work progressively
- this keeps the source fetch stage compatible with future worker parallelism without changing the source contract

### Best-effort dedupe policy

Primary rule:

- dedupe by source-provided `unique_identifier`

Important limits:

- this is best effort only
- not all duplicate real-world images will be detected
- correctness must not depend on dedupe being perfect

Recommended lookup flow:

1. source iterator yields image metadata
2. source implementation generates `unique_identifier`
3. runner checks whether an `images` row already exists
4. if found, reuse local canonical file instead of redownloading when possible
5. if a device-visible target path is needed, create hard link first
6. if hard link fails, fall back to copy

### Hard-link strategy

Target behavior:

- avoid redownload when local image content already exists
- avoid copy when hard link is possible
- use copy only as fallback

Important operational notes:

- hard links require the same filesystem or volume
- Windows, Linux, and macOS all support hard links, but permissions and filesystem behavior differ
- Docker bind mounts and cross-filesystem targets may force fallback copy behavior

Recommended storage layout:

- keep all device image paths under the same data directory when possible
- store device image files with this pattern:

```text
{base_dir}/images/{device_slug}/{image_unique_identifier}.{ext}
```

- this layout is intentional so syncing tools such as `rsync` and `syncthing` can mirror device-specific directories easily
- image location tracking is required because one image can exist in many device-specific paths
- same-filesystem placement still maximizes chance that hard-linking works across supported deployments

### Thumbnail generation

Thumbnail generation should happen as a standard post-download step.

Goals:

- keep UI image listing lightweight
- avoid loading original multi-megabyte files for grid/list views
- ensure one thumbnail per stored image for the common case

Recommended behavior:

1. after canonical image file is available, check whether thumbnail already exists
2. if missing, generate thumbnail from the canonical image
3. store thumbnail metadata and path in `image_thumbnails`
4. serve thumbnail in UI list/gallery responses when available

Implementation constraints:

- thumbnail generation library must remain pure Go
- no CGO, native bindings, or FFI
- deletion of the parent image should also clean up thumbnail artifacts

### Delete semantics

When user deletes an image:

1. load all `image_locations` rows for that image
2. attempt to delete all tracked file paths on disk
3. remove or deactivate successful location rows
4. remove the `images` row when tracked locations are fully cleaned
5. if cleanup is partial, return detailed failure information and keep remaining records consistent

This ensures tracked hard links and copied locations can be cleaned together.

### Concurrency policy

Keep this intentionally simple at first.

Recommendation:

- fixed worker count, default `1`
- optional config to raise concurrency later
- assignment logic should still treat one job as belonging to one `sources` row with many subscribed candidate devices

This preserves single-process simplicity while preventing early race complexity around SQLite writes.

## Filtering Rules Plan

For device image selection:

1. Image must come from a source the device subscribes to.
2. If device `is_adult_allowed = false`, image with `is_adult = true` must be excluded.
3. Image aspect ratio must fall within device tolerance.
4. Image resolution should be at least device size unless an optional fallback mode is introduced later.
5. Device explicit min/max image dimension and filesize bounds must be respected when set.
6. Source and subscription must both be enabled.
7. Tag filtering should be supported through relational tag tables, not JSON arrays.

Suggested SQL-level checks:

- `image.width >= device.screen_width`
- `image.height >= device.screen_height`
- `ABS(image.aspect_ratio - CAST(device.screen_width AS REAL) / device.screen_height) <= device.aspect_ratio_tolerance`
- `(device.min_image_width = 0 OR image.width >= device.min_image_width)`
- `(device.max_image_width = 0 OR image.width <= device.max_image_width)`
- `(device.min_image_height = 0 OR image.height >= device.min_image_height)`
- `(device.max_image_height = 0 OR image.height <= device.max_image_height)`
- `(device.min_filesize = 0 OR image.file_size_bytes >= device.min_filesize)`
- `(device.max_filesize = 0 OR image.file_size_bytes <= device.max_filesize)`

This filtering should happen in SQL where possible.

## Image Query and Favorite Plan

Image listing should support operational browsing, cleanup, and search.

### Image list filters

Required filters:

- adult content flag
- file size range
- image width range
- image height range
- search by uploader
- search by artist
- search by origin URL
- search by original source item ID
- favorite status

### Metadata requirements

To support those filters cleanly, `images` should be extended with searchable metadata fields.

Recommended additional columns on `images`:

- `uploader` TEXT
- `artist` TEXT
- `origin_url` TEXT
- `original_id` TEXT
- `width` INTEGER
- `height` INTEGER
- `is_adult` INTEGER not null default 0
- `is_favorite` INTEGER not null default 0

Notes:

- image metadata should live directly on `images` because there is no separate discovered-wallpaper phase

### Favorite behavior

- favorites are stored on `images`
- favorite is a manual user flag
- favorite survives source resync unless the image itself is deleted
- image list should support filtering `favorite only`

### Search behavior

For phase 1:

- use SQLite `LIKE` search on uploader, artist, origin URL, and original ID
- add indexes later based on observed query behavior

## Backend API Plan

### Handler and service file structure

Recommended backend structure:

- `internal/routes/register.go`
  - central place to register all Huma handlers
- `internal/routes/[domain]/{MethodName}.go`
  - one file per RPC method
  - exposes one `*huma.Operation`
  - exposes the route handler function registered from `register.go`
  - owns route-layer request/response types when composition or aliasing is needed
  - acts only as request entrypoint and delegates to services
- `internal/services/[domain]/[domain].go`
  - base service struct and dependencies
- `internal/services/[domain]/{MethodName}.go`
  - method receiver implementation and request/response types

### Route-layer responsibilities

Routes should stay thin.

- define Huma operation metadata
- define route request/response shape when needed
- map route DTOs to service request/response types
- call the relevant service method
- return the service result

Routes should not:

- hold business logic
- perform direct SQL operations beyond trivial composition needs
- duplicate service-layer filtering or orchestration logic

### Service-layer responsibilities

Services should own application behavior.

- query and mutate database
- enforce business rules
- coordinate scheduler, queue, runner, and storage behavior
- define method-level request/response contracts

### Example service structure

`internal/services/images/images.go`

```go
type ImageService struct {
    db *sql.DB
}
```

`internal/services/images/ListImages.go`

```go
func (service ImageService) ListImages(ctx context.Context, req ListImagesRequest) (res ListImagesResponse, err error)

type ListImagesRequest struct {
    DeviceIDs []uuid.UUID `json:"device_ids" doc:"Filter images assigned to the device ids if not empty"`
    CursorPaginationRequest
}

type ListImagesResponse struct {
    Items []models.Images `json:"items" doc:"List of images matching the filters"`
    Total uint64 `json:"total" doc:"Total number of images matching the filters"`
    CursorPaginationResponse
}
```

### DTO composition rule

At the route layer:

- use pure aliasing to service request/response types when the route contract matches exactly
- use composed route DTOs only when Huma-specific wrapping or transport-specific differences are needed

This keeps service contracts reusable while allowing route-specific metadata when required.

### Shared pagination contracts

Recommended shared pagination types:

```go
type CursorPaginationRequest struct {
    Next   *uuid.UUID `json:"next" doc:"Return items after this cursor if provided"`
    Prev   *uuid.UUID `json:"prev" doc:"Return items before this cursor if provided"`
    Offset uint64     `json:"offset" doc:"Maximum number of items to return from the cursor position"`
}

type CursorPaginationResponse struct {
    Next *uuid.UUID `json:"next" doc:"Cursor for the next page if more items are available"`
    Prev *uuid.UUID `json:"prev" doc:"Cursor for the previous page if available"`
}
```

Rules:

- list-style RPC methods should prefer this shared cursor pagination shape
- `Next` and `Prev` are opaque UUID cursors from the API consumer point of view
- `Offset` acts as page size / limit for the current request

### Route prefix and base path rules

Walens should support a configurable deployment base path.

Rules:

- default base path is `/`
- user can configure another base path, for example `/walens`
- frontend asset serving, SPA fallback, login page, and API routes must all honor the configured base path
- if base path is `/walens`, API routes become `/walens/api/...`
- Vite dev mounting and production asset serving must both work correctly when base path is not `/`

Recommended config:

- `server.base_path`

Examples:

- base path `/` -> API prefix `/api`
- base path `/walens` -> API prefix `/walens/api`

### RPC routing convention

Backend API should use RPC-style route naming.

Pattern:

- `{prefix}/{version}/{domain}/{MethodName}`

Where:

- `{prefix}` is `/api` under the configured base path
- `{version}` starts with `v1`
- `{domain}` is the logical service area, for example `images`, `devices`, `sources`
- `{MethodName}` is a PascalCase RPC-style operation name

Examples:

- `POST /api/v1/images/ListImages`
- `POST /walens/api/v1/images/ListImages`

### HTTP method convention

Rules:

- all normal backend methods are `POST`
- all normal backend methods use JSON request body
- this includes list, get, search, delete, favorite, and sync-style operations
- only explicit exceptions such as health check, scalar/openapi UI, static assets, login page delivery, and similar infrastructure routes may use non-`POST` methods

Rationale:

- keeps backend API convention uniform
- aligns with RPC-style naming
- avoids route-shape drift between read and write operations

### Auth endpoints

- `GET {base}/login`
  - serves simple login page when auth is enabled
- `POST {base}/auth/login`
  - validates configured credentials and sets HTTP-only cookie for browser use
- `POST {base}/auth/logout`
  - clears auth cookie

### Documentation endpoints

- `GET {base}/docs`
  - hosts Scalar UI
- `GET {base}/openapi.json`
  - hosts OpenAPI JSON
- `GET {base}/openapi.yaml`
  - hosts OpenAPI YAML

Base-path examples:

- base path `/` -> `/docs`, `/openapi.json`, `/openapi.yaml`
- base path `/walens` -> `/walens/docs`, `/walens/openapi.json`, `/walens/openapi.yaml`

Important rule:

- these documentation endpoints are always mounted under the configured base path prefix and must never assume root-only deployment

### API methods by domain

All examples below assume base path `/`.

#### Devices

- `POST /api/v1/devices/ListDevices`
- `POST /api/v1/devices/GetDevice`
- `POST /api/v1/devices/CreateDevice`
- `POST /api/v1/devices/UpdateDevice`
- `POST /api/v1/devices/DeleteDevice`

#### Configs

- `POST /api/v1/configs/GetConfig`
- `POST /api/v1/configs/UpdateConfig`
  - replaces the full persisted config object atomically

#### Source Types

- `POST /api/v1/source_types/ListSourceTypes`
- `POST /api/v1/source_types/GetSourceType`
  - returns validation/schema metadata for one source type, with params exposed as JSON Schema via `*huma.Schema`

#### Sources

- `POST /api/v1/sources/ListSources`
- `POST /api/v1/sources/GetSource`
- `POST /api/v1/sources/CreateSource`
- `POST /api/v1/sources/UpdateSource`
- `POST /api/v1/sources/DeleteSource`
- `POST /api/v1/sources/SyncSource`
  - creates manual source sync job and enqueues it

#### Source Schedules

- `POST /api/v1/source_schedules/ListSourceSchedules`
- `POST /api/v1/source_schedules/CreateSourceSchedule`
- `POST /api/v1/source_schedules/UpdateSourceSchedule`
- `POST /api/v1/source_schedules/DeleteSourceSchedule`

#### Device Subscriptions

- `POST /api/v1/device_subscriptions/ListDeviceSubscriptions`
- `POST /api/v1/device_subscriptions/CreateDeviceSubscription`
- `POST /api/v1/device_subscriptions/UpdateDeviceSubscription`
- `POST /api/v1/device_subscriptions/DeleteDeviceSubscription`

#### Images

- `POST /api/v1/images/ListImages`
  - supports filters for adult flag, favorite, file size, width, height, uploader, artist, origin URL, and original ID
- `POST /api/v1/images/GetImage`
- `POST /api/v1/images/ListDeviceImages`
  - returns images matching device filters and device subscriptions
- `POST /api/v1/images/SetImageFavorite`
- `POST /api/v1/images/BlacklistImage`
  - blacklists an image by source-specific unique identifier so it is never downloaded again
- `POST /api/v1/images/DeleteImage`
  - best-effort delete all tracked disk locations and then remove the image record
- `POST /api/v1/images/GetImageThumbnail`
  - returns or resolves thumbnail access information for UI usage

#### Jobs

- `POST /api/v1/jobs/ListJobs`
- `POST /api/v1/jobs/GetJob`

#### Admin / Runtime

- `POST /api/v1/admin/ReloadScheduler`
  - optional debug/admin method
- `POST /api/v1/admin/GetRuntimeStatus`
  - optional runtime status including queue size, scheduler state, workers

### OpenAPI strategy

- Huma owns request/response types and OpenAPI generation.
- Frontend client is generated from the emitted OpenAPI document.
- DTOs stay separate from database models.
- Database field documentation should be descriptive enough that generated model metadata can later enrich API docs where appropriate.

## Auth Plan

### Config model

Recommended auth config:

- `auth.enabled` boolean
- `auth.username` string
- `auth.password` string
- values can come from env vars or config file

Separation rule:

- auth config is separate from the persisted `configs` table
- auth config should not be stored in database config
- auth credentials remain bootstrap/runtime settings only

Rules:

- if `auth.enabled = false`, no auth middleware is enforced
- if `auth.enabled = true`, both frontend and API routes should be protected except the login route and static assets required to render it
- credentials are read-only at runtime from config/env and are not editable via UI or API
- there is no user table and no account model in the application

### Request auth behavior

When auth is enabled:

1. inspect `Authorization` header for Basic credentials
2. if header is missing, inspect HTTP-only auth cookie
3. validate against configured username/password
4. reject with `401` when invalid

Recommended browser behavior:

- browser login page submits credentials to a login endpoint
- server validates credentials and sets an HTTP-only cookie
- subsequent browser requests can authenticate via cookie
- login page route should also honor configured base path, for example `/walens/login`

Recommended API/client behavior:

- non-browser clients and future mobile app use `Authorization` header directly

### Cookie/session model

Keep this intentionally simple.

Recommendation:

- use signed or opaque cookie value derived from configured credentials or server-side auth secret
- mark cookie as HTTP-only
- use `Secure` when served over HTTPS
- add `SameSite` policy appropriate for same-host deployment

Since the target deployment is same-host frontend + backend, a minimal cookie model is acceptable.

Important scope boundary:

- this is not login tied to per-user identity
- this is only a thin protection layer in front of a single-user/self-hosted app

### Login UI behavior

- if auth is enabled and browser request has no valid cookie, show simple login page
- login page is only for browser/self-hosted UI access
- after successful login, redirect to app shell

### Logout behavior

- provide a simple logout endpoint that clears auth cookie
- this is optional for phase 1 UI but recommended

## Go-Jet Customization Plan

Requirements:

1. Integer columns with prefix `is_` map to custom boolean type.
2. Columns with prefix `json_` and db type `json`, `jsonb`, or `string/text` map to custom `json.RawMessage` wrapper.
3. Integer columns ending with `_ms` and representing durations map to a custom duration type backed by milliseconds.
4. Timestamp fields such as `created_at`, `updated_at`, `run_after`, `started_at`, and `finished_at` map to a custom `time.Time` wrapper backed by Unix milliseconds.
5. Columns named exactly `id` or ending with `_id` map to `uuid.UUID` from `google/uuid`.
6. Internal database IDs are generated by the app as UUIDv7 values.
7. External IDs or source-specific unique identifiers should use `identifier` naming instead of `id` naming to avoid conflict with internal IDs.
8. Column comments generate `doc:"${comment}"` tag if not empty.
9. Generated models also include `json:"${column_name}"` tag.

Implementation plan:

- Add a codegen wrapper around Jet generation rather than editing generated files.
- Pipeline:
  1. run migrations against a local dev SQLite database
  2. run Jet generator against that schema
  3. post-process generated model structs with Go AST rewrite
- Prefer AST rewrite over regex.

Suggested custom types:

- `type BoolInt bool`
- `type RawJSON json.RawMessage`
- `type UnixMilliDuration time.Duration`
- `type UnixMilliTime time.Time`

OpenAPI/schema requirement for custom types:

- custom types used in request/response models should implement Huma's `SchemaProvider` interface
- this is required so generated OpenAPI metadata remains accurate for wrapped boolean, JSON, UUID-adjacent, and Unix-millisecond time representations
- schema output should describe the transport shape, not only the internal Go wrapper type
- for timestamp wrappers, `SchemaProvider` should point to the same schema shape as `time.Time`, so transport stays RFC3339Nano string even if database storage uses Unix milliseconds
- for duration wrappers, `SchemaProvider` should describe integer milliseconds

Duration mapping rules:

- integer columns ending with `_ms` and representing durations are stored in SQLite as INTEGER milliseconds
- generated model fields should use a custom duration wrapper type rather than raw integer
- when writing to the database, the wrapper should persist duration as integer milliseconds
- when reading from the database, the wrapper should parse integer milliseconds into Go duration value
- JSON output should be integer milliseconds
- JSON input should accept either duration string or integer milliseconds
- when JSON input is integer, treat it as milliseconds
- current planned example field: `duration_ms`

Time mapping rules:

- timestamp columns are stored in SQLite as INTEGER Unix milliseconds
- generated model fields should use a custom time wrapper type rather than raw integer
- when writing to the database, the wrapper should persist Unix milliseconds
- when reading from the database, the wrapper should parse Unix milliseconds into time value
- JSON transport should reuse normal `time.Time` behavior rather than custom JSON marshaling logic
- this applies to standard audit fields such as `created_at` and `updated_at` and also runtime timestamps such as `run_after`, `started_at`, and `finished_at`

ID mapping rules:

- database columns named `id` map to `uuid.UUID`
- database columns ending in `_id` map to `uuid.UUID`
- this includes foreign keys such as `source_id`, `device_id`, `tag_id`, `image_id`, and similar fields
- all internal IDs are UUIDv7 values generated by the application layer

Naming rules for external identifiers:

- use `*_identifier` for external or source-owned identifiers
- examples: `source_item_identifier`, `original_identifier`
- reserve `id` and `*_id` for internal database identity and internal relations only

Important note:

- SQLite does not natively expose rich column comments like PostgreSQL/MySQL.
- If `doc` tags must come from column comments, Walens likely needs a sidecar schema metadata file or migration annotations consumed by the codegen wrapper.
- This remains one of the highest-risk implementation spikes.

### Schema documentation requirement

Every migration-defined field should carry documentation metadata.

Expectation:

- each table column should have a descriptive comment
- comments should explain the business meaning of the field, not just restate the name
- this documentation should flow into generated Jet model tags via `doc:"..."`
- that generated metadata should be usable later to enrich OpenAPI-facing DTO documentation
- field naming should clearly separate internal IDs from external identifiers for codegen clarity and API readability

Practical implication for SQLite:

- since native column comments are weak or unavailable, Walens should plan for a schema metadata sidecar or migration annotation format as the source of truth for field documentation

## Database and Migration Plan

### Persisted app config table

Walens should support persisted application config in the database.

Recommended table:

- `configs`

Suggested columns:

- `id` INTEGER primary key
- `value` TEXT not null
- `updated_at` INTEGER not null

Semantics:

- `value` stores the full serialized config struct as JSON
- `value` is replaced atomically on update, not patched field-by-field in SQL
- Walens should treat this table as a single-row config store in practice
- when the table is empty during boot, Walens should insert the default config immediately
- application config should be managed from this table only, not through layered app-config sources
- auth settings and similar bootstrap/runtime-only settings should stay outside this table

Recommended operational rule:

- startup loads the config row if present
- if absent, startup inserts default values and uses them as active config
- config writes replace the entire `value` column in one atomic operation
- prefer simple consistency over config caching complexity; reading the config row again for relevant operations is acceptable

Note for SQLite:

- although the requirement says `jsonb`, SQLite should store this effectively as JSON text while keeping the app-side config struct behavior the same

### Config struct serialization

The app config struct should implement both database serialization interfaces.

Required behavior:

- implement `driver.Valuer` using `json.Marshal(self)`
- implement `driver.Scanner` using `json.Unmarshal(...)`
- implement Huma `SchemaProvider` when the type is exposed through API models

Benefits:

- one canonical config representation in Go
- direct persistence into the `configs.value` column
- easy whole-object replacement on update

The custom timestamp wrapper should follow the same database conversion pattern.

Required behavior:

- implement `driver.Valuer` by returning Unix milliseconds
- implement `driver.Scanner` by reading INTEGER milliseconds from the database and converting to Go time
- no custom JSON marshaler/unmarshaler is required if the type preserves normal `time.Time` JSON behavior
- implement Huma `SchemaProvider` so OpenAPI points to the same schema shape as `time.Time`

### Config service and routes

Walens needs dedicated config routes and services.

Recommended files:

- `internal/routes/configs/GetConfig.go`
- `internal/routes/configs/UpdateConfig.go`
- `internal/services/configs/configs.go`
- `internal/services/configs/GetConfig.go`
- `internal/services/configs/UpdateConfig.go`

Recommended service shape:

```go
type ConfigService struct {
    db *sql.DB
}
```

Expected behavior:

- `GetConfig` returns the active persisted config object
- `UpdateConfig` replaces the entire config value atomically
- if config row does not exist yet, service may initialize it from defaults before returning or updating
- auth config is not managed by these routes/services

### SQLite driver

Use a pure-Go driver:

- `modernc.org/sqlite`

This satisfies the no-CGO requirement and simplifies static builds across all supported OS/arch targets.

### Migrations

Use `golang-migrate` with SQLite driver.

Plan:

- keep SQL migrations in `internal/db/migrations`
- embed migrations into binary for production use
- execute migrations automatically on startup
- optionally add a migrate-only CLI mode later

Migration ordering rule:

- the very first migration should contain SQLite optimizations and best-practice setup only
- that first migration is the place for SQLite-specific baseline setup that should exist before business tables
- business/domain schema migrations should start from the second migration onward

Recommended split:

- migration 1: SQLite setup and operational best practices
- migration 2+: configs, devices, sources, schedules, subscriptions, images, tags, image_tags, image_assignments, image_locations, image_thumbnails, image_blacklists, jobs, and later business schema changes

### SQLite pragmas

At startup apply pragmatic defaults:

- `foreign_keys = ON`
- `busy_timeout`
- `journal_mode = WAL` when appropriate

Need to validate WAL behavior across:

- Linux host volume
- macOS host filesystem
- Windows local deployment
- Docker mounted volume

## Frontend Plan

### SPA structure

Frontend should be designed mobile first.

Core requirement:

- the UI must feel smooth and usable on mobile devices first, then scale upward to larger viewports

Primary screens:

1. Device list page
2. Device detail page
3. Source list page
4. Source detail page with schedules
5. Subscription management page
6. Job history page
7. Wallpaper results page per device

### Data access

- Generate OpenAPI types from Huma spec.
- Use `openapi-fetch` to create typed client.
- Wrap all remote calls in TanStack Svelte Query.

### UI behavior

- Device form captures dimensions, adult toggle, aspect tolerance, and explicit min/max image dimension/filesize constraints.
- Source form selects source implementation type and edits params JSON.
- Source detail page manages many cron schedules.
- Job page shows status, run time, duration, counts, and error reason.
- image list/gallery should use masonry layout for previews
- mobile phone view should render at least 2 masonry columns
- wider viewports should dynamically increase column count based on available width
- gallery/list views should stay smooth on mobile while browsing many images
- image grid shows preview, source, resolution, aspect ratio, and adult badge.
- image list/gallery views should prefer generated thumbnails over original files
- Source param UI should be able to evolve from raw JSON editing into schema-driven forms using `*huma.Schema` / JSON Schema.
- when auth is enabled, browser users without valid auth cookie should see a simple login page before entering the app

### Frontend performance rule

- frontend components should be as lazy-loaded as possible
- components that pull heavy dependencies or expensive rendering logic must be lazy
- image-heavy views should avoid eager rendering of non-visible content when possible
- mobile performance is a first-class requirement, not an afterthought

### Initial simplification

- Use validated JSON textarea for source params before building schema-driven forms.
- Show schedule proximity warnings as plain UI alerts first.
- Keep login UI intentionally minimal: username, password, submit, and invalid-credential message.

## Dev Server Integration Plan

Using `github.com/olivere/vite`:

- in development, Go server mounts or proxies Vite dev server assets
- in production, Go server serves built frontend assets from embedded files or packaged static directory
- API and frontend remain same-origin
- configured base path must be honored by both dev and prod asset serving
- Vite integration must work correctly when app is mounted under non-root paths such as `/walens`

Recommended route split:

- `{base}/api/*` served by Huma
- `{base}/login` served by login UI when auth is enabled
- `{base}/docs`, `{base}/openapi.json`, and `{base}/openapi.yaml` served as documentation endpoints
- other `{base}/*` non-API routes served by SPA fallback

Important note:

- frontend asset URLs, router base, and backend-mounted Vite asset paths must all be derived from the same configured base path to avoid broken assets when deploying under a subpath

## Build and Deployment Plan

### Binary output

- build with `CGO_ENABLED=0`
- use only pure-Go backend dependencies
- embed migrations and production frontend assets where possible
- produce release artifacts for all 6 target OS/arch combinations

### Docker output

Multi-stage Dockerfile:

1. frontend build stage
2. Go build stage
3. minimal runtime stage containing only binary and data directory setup

Recommended target outcome:

- `docker run walens` starts the full app
- database is stored on mounted volume
- no sidecar services required
- same binary behavior as non-container deployment

### Config and filesystem plan

Recommended runtime config:

- application config lives in the persisted database config row
- one data directory for SQLite and future app-managed files
- separate bootstrap/runtime auth toggle with static username/password credentials
- configurable server base path for subpath deployment

Config management rule:

- use the `configs` table as the single source of truth for application config
- when `configs` is empty, startup should insert the default config into the table
- updates replace the full config value atomically
- avoid layered app-config merging logic because it adds complexity with low payoff
- do not mix auth credentials into persisted application config

Recommended bootstrap/runtime auth fields:

- `WALENS_AUTH_ENABLED`
- `WALENS_AUTH_USERNAME`
- `WALENS_AUTH_PASSWORD`

Recommended bootstrap/runtime server path fields:

- `WALENS_SERVER_BASE_PATH`

Examples:

- Linux/macOS binary: `./walens --data-dir ./data`
- Windows binary: `walens.exe --data-dir .\data`
- Docker: mount `/data`

## Execution Roadmap

### Phase 0 - Validation spikes

1. Verify Huma + standard `net/http` bootstrapping.
2. Verify `olivere/vite` integration with SvelteKit SPA.
3. Verify `modernc.org/sqlite` + `golang-migrate` compatibility.
4. Verify Go-Jet generation workflow against SQLite schema and confirm customization path.
5. Verify cross-compilation and release packaging for 6 targets.
6. Verify cron parser/library is pure Go and suitable for reloadable schedules.
7. Verify base-path mounting works for API routes, login page, SPA fallback, and Vite-served assets.

### Phase 1 - Runtime skeleton

1. Create app bootstrap, config, logging, database connection, and health endpoint.
2. Add auth config loading and optional auth middleware.
3. Add migration runner.
4. Add runtime manager for HTTP server, scheduler, queue, and job runner in one process.
5. Add first migration for SQLite optimizations and best practices.
6. Add business schema migrations for configs, devices, sources, source schedules, subscriptions, images, tags, image_tags, image_assignments, image_locations, image_thumbnails, image_blacklists, and jobs.
7. Add persisted config bootstrapping with default row injection.
8. Add Jet generation script/tooling.

### Phase 2 - Domain and API

1. Implement config repository/service/API.
2. Implement device repository/service/API.
3. Implement source registry and source-type metadata API.
4. Implement configured source repository/service/API.
5. Implement source schedule repository/service/API with cron validation and warning detection.
6. Implement device subscription API.

### Phase 3 - Jobs and Scheduling

1. Implement persisted jobs table and repository.
2. Implement in-memory queue.
3. Implement job runner with boot recovery.
4. Implement scheduler reload behavior on source/schedule mutations.
5. Add manual sync endpoint.

### Phase 4 - Ingestion and Wallpapers

1. Build normalized image metadata model.
2. Implement one source first, ideally `BooruSource`.
3. Persist image metadata from jobs.
4. Implement image dedupe, assignment tracking, device-oriented storage, image location tracking, blacklist checks, and thumbnail generation.
5. Implement per-device filtered image query endpoint.

### Phase 5 - Frontend SPA

1. Scaffold SvelteKit SPA.
2. Integrate generated API client.
3. Build simple login UI and cookie-based browser auth flow.
4. Build devices UI.
5. Build source and schedule UI.
6. Build subscriptions UI.
7. Build jobs UI.
8. Build mobile-first image gallery UI with responsive masonry layout.
9. Add lazy-loading for heavy frontend components and image-heavy views.
10. Build image management UI with search, filter, favorite, and delete flow.

### Phase 6 - Packaging

1. Embed/build production frontend assets.
2. Produce static Go binaries for all target platforms.
3. Produce single-container deployment.
4. Document local dev, binary deployment, and Docker deployment workflow.

## Risks and Open Questions

### 1. SQLite column comments

SQLite does not support column comments in the same rich way as PostgreSQL/MySQL.

Impact:

- `doc` tag generation cannot rely on normal SQLite schema reflection alone.

Mitigation options:

- sidecar schema manifest used by codegen
- migration annotations parsed by custom codegen
- custom model metadata layer outside Jet

Recommendation:

- treat field documentation as required schema metadata from the start
- write descriptive comments for every migration field
- do not rely on SQLite reflection alone to recover those comments later

### 2. Cron overlap warning complexity

Cron expressions can create non-obvious overlaps.

Recommendation:

- use bounded lookahead validation for the next 14 days
- classify result as warning, not hard error, for phase 1

### 3. SQLite write contention

Single-process runtime reduces complexity, but scheduler/job runner/API can still contend for writes.

Recommendation:

- start with low worker concurrency
- keep writes short and transactional
- monitor busy timeout behavior early

### 4. Frontend dependency purity

Some frontend toolchain packages may indirectly pull optional native helpers.

Recommendation:

- evaluate additions carefully and keep the default toolchain minimal
- prefer portable JS tooling without native compile steps

### 5. Asset storage strategy

Walens now needs local downloaded file tracking with best-effort dedupe.

Recommendation:

- store device-oriented downloaded files locally
- track assignment state in `image_assignments`
- track all derived file paths in `image_locations`
- maintain image blacklist state in `image_blacklists`
- generate and track lightweight thumbnails for UI use
- use hard links first and copy as fallback
- treat dedupe as optimization, not correctness guarantee

### 6. Cross-platform hard-link behavior

Hard-link support exists on target OSes but can vary by filesystem and mount layout.

Recommendation:

- explicitly test hard-link create/delete behavior on Windows, Linux, macOS, and Docker bind mounts
- keep copy fallback as a first-class supported path

### 7. Basic Auth simplicity vs security

This auth layer is intentionally simple and should be treated as deployment protection, not a full identity system.

Recommendation:

- keep credential handling simple and explicit
- document that credentials are configured outside the app and not editable from UI
- focus on reducing casual abuse, crawler access, and exposure of adult content rather than advanced threat resistance
- do not expand this into user management unless product scope changes substantially

## Recommended First Implementation Order

If implementation starts now, the best order is:

1. bootstrap one-process runtime with Go server + SQLite + migrations
2. validate pure-Go dependency set and cross-platform build matrix
3. validate Jet customization feasibility
4. implement config schema/service/API
5. implement sources + schedules schema/API
6. implement scheduler reload and warning detection
7. implement jobs persistence + in-memory queue + runner
8. implement images + image_assignments + image_locations + image_thumbnails + image_blacklists schema and storage flow
9. implement device and subscription schema/API
10. implement image filtering query APIs
11. scaffold SPA and connect typed client
12. add first concrete source ingestion

## Deliverables for the Next Step

After planning, the next concrete implementation package should ideally produce:

- runnable one-process Go server skeleton
- first migration for SQLite setup and best practices
- migration set for configs, devices, sources, schedules, subscriptions, images, tags, image_tags, image_assignments, image_locations, image_thumbnails, image_blacklists, and jobs
- Jet generation script/prototype
- scheduler/queue/runner skeleton wired into app lifecycle
- Huma OpenAPI output
- SvelteKit SPA scaffold integrated with Vite mounting
- first config, device, source, schedule, subscription, and image listing/favorite endpoints
