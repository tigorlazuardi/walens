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

5. **Wallpaper**
   - Persisted normalized wallpaper metadata catalog.
   - Used to answer filtered wallpaper queries for devices.

6. **Image**
   - Persisted downloaded image identity record.
   - Uses source-provided `unique_id` for best-effort dedupe.
   - Represents canonical local image content when a file has already been downloaded.

7. **Image Location**
   - Persisted tracker for every file path on disk that points to an image.
   - Supports hard-link tracking and cleanup.

8. **Job**
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
- Ingest wallpapers from source-row jobs into local SQLite database.
- Perform best-effort dedupe on downloaded images using source-provided `unique_id`.
- Reuse existing local image files across devices without redownload when possible.
- Prefer hard link creation for additional device-visible paths, then fallback to file copy.
- Track all image file locations on disk for later cleanup.
- Query wallpapers for a specific device using device filters and subscribed sources.

### Explicit non-goals for now

- High availability.
- Multi-node scheduling.
- External queue broker.
- Distributed workers.
- User accounts and auth.
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
    domain/
      device/
      source/
      schedule/
      subscription/
      wallpaper/
      job/
    scheduler/
    queue/
    runner/
    ingest/
    http/
      api/
      dto/
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
- `screen_width` INTEGER not null
- `screen_height` INTEGER not null
- `is_adult_allowed` INTEGER not null default 0
- `aspect_ratio_tolerance` REAL not null default 0.15
- `created_at` TEXT not null
- `updated_at` TEXT not null

Notes:

- `aspect_ratio_tolerance` should be absolute ratio delta tolerance.
- Example: device ratio `16/9 ~= 1.777`, wallpaper ratio accepted if `abs(wallpaper_ratio - device_ratio) <= tolerance`.

### 2. `sources`

This is now a first-class table.

Suggested columns:

- `id` TEXT primary key
- `name` TEXT not null
- `source_type` TEXT not null
- `params` TEXT not null default '{}'
- `lookup_count` INTEGER not null default 0
- `is_enabled` INTEGER not null default 1
- `created_at` TEXT not null
- `updated_at` TEXT not null

Semantics:

- `name` is the user-defined unique name of the source row in `sources`.
- `source_type` is the code-registered source implementation name, for example `booru` or `reddit`.
- `params` stores stringified JSON for that specific source row.
- `lookup_count` controls how many candidate images the source row should try to fetch/look up per run.
- if `lookup_count = 0`, Walens should use the default from the source implementation.
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
- `created_at` TEXT not null
- `updated_at` TEXT not null

Notes:

- Cron format should be standardized early, preferably 5-field cron unless there is a strong reason to support seconds.
- Validation should happen at API boundary and in domain service.

### 4. `device_source_subscriptions`

Suggested columns:

- `id` TEXT primary key
- `device_id` TEXT not null references `devices(id)` on delete cascade
- `source_id` TEXT not null references `sources(id)` on delete cascade
- `is_enabled` INTEGER not null default 1
- `created_at` TEXT not null
- `updated_at` TEXT not null

Constraints:

- unique index on `(device_id, source_id)`

### 5. `wallpapers`

Suggested columns:

- `id` TEXT primary key
- `source_id` TEXT not null references `sources(id)` on delete cascade
- `source_type` TEXT not null
- `source_item_id` TEXT not null
- `original_url` TEXT not null
- `preview_url` TEXT
- `width` INTEGER not null
- `height` INTEGER not null
- `aspect_ratio` REAL not null
- `is_adult` INTEGER not null default 0
- `json_tags` TEXT not null default '[]'
- `json_meta` TEXT not null default '{}'
- `created_at` TEXT not null
- `updated_at` TEXT not null

Constraints:

- unique index on `(source_id, source_item_id)`
- optional dedupe index on `original_url` later if source stability is confirmed

### 6. `images`

This table stores canonical downloaded image identity records.

Suggested columns:

- `id` TEXT primary key
- `source_id` TEXT references `sources(id)`
- `wallpaper_id` TEXT references `wallpapers(id)`
- `unique_id` TEXT not null
- `original_filename` TEXT
- `origin_url` TEXT
- `original_id` TEXT
- `uploader` TEXT
- `artist` TEXT
- `mime_type` TEXT
- `file_size_bytes` INTEGER
- `width` INTEGER
- `height` INTEGER
- `is_adult` INTEGER not null default 0
- `is_favorite` INTEGER not null default 0
- `json_meta` TEXT not null default '{}'
- `created_at` TEXT not null
- `updated_at` TEXT not null

Semantics:

- `unique_id` is generated by the source implementation.
- `unique_id` is best effort only.
- duplicates can still exist in reality if the source emits unstable IDs or different sources describe the same file differently.

Constraints:

- unique index on `(source_id, unique_id)` is recommended for phase 1.

### 7. `image_locations`

This table tracks every concrete file path on disk for an image.

Suggested columns:

- `id` TEXT primary key
- `image_id` TEXT not null references `images(id)` on delete cascade
- `device_id` TEXT references `devices(id)`
- `path` TEXT not null
- `storage_kind` TEXT not null
- `is_primary` INTEGER not null default 0
- `is_active` INTEGER not null default 1
- `created_at` TEXT not null
- `updated_at` TEXT not null

Suggested enum-like values:

- `storage_kind`: `canonical`, `hardlink`, `copy`

Semantics:

- `canonical` is the first locally stored file path for the image.
- `hardlink` is an additional filesystem path pointing to the same content through a hard link.
- `copy` is fallback storage when hard links cannot be created.
- `device_id` is optional because some files may be shared/global rather than tied to one device path.

Constraints:

- unique index on `path`

### 8. `jobs`

Jobs are first-class persisted records.

Suggested columns:

- `id` TEXT primary key
- `job_type` TEXT not null
- `source_id` TEXT references `sources(id)`
- `source_name` TEXT
- `source_type` TEXT
- `status` TEXT not null
- `trigger_kind` TEXT not null
- `run_after` TEXT not null
- `started_at` TEXT
- `finished_at` TEXT
- `duration_ms` INTEGER
- `requested_image_count` INTEGER not null default 0
- `downloaded_image_count` INTEGER not null default 0
- `reused_image_count` INTEGER not null default 0
- `hardlinked_image_count` INTEGER not null default 0
- `copied_image_count` INTEGER not null default 0
- `stored_wallpaper_count` INTEGER not null default 0
- `error_message` TEXT
- `json_input` TEXT not null default '{}'
- `json_result` TEXT not null default '{}'
- `created_at` TEXT not null
- `updated_at` TEXT not null

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
- `error_message` stores summarized terminal failure reason.
- `json_input` stores source params / execution request snapshot.
- `json_result` stores extensible metadata such as source cursor, warnings, or counts by category.

### 9. `job_attempts` (optional)

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
- sync/fetch logic that returns normalized wallpaper records
- best-effort `unique_id` generation for downloadable image items

Suggested conceptual interface:

```go
type Source interface {
    TypeName() string
    DisplayName() string
    ValidateParams(raw json.RawMessage) error
    ParamSchema() *huma.Schema
    DefaultLookupCount() int
    Sync(ctx context.Context, req SyncRequest) (*SyncResult, error)
    BuildUniqueID(item SourceImageItem) (string, error)
}
```

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
- the runner executes the `booru` implementation with params from `anime-pics-main`
- devices subscribed to `anime-pics-main` are the candidate set for downstream image fetching and assignment decisions

Download/storage meaning:

- if the source implementation returns an item whose `unique_id` already exists locally, Walens should reuse the existing file when possible
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
- Invoke source implementation.
- Persist results, counts, errors, and timings.
- Store discovered wallpapers.
- Resolve or create `images` rows using source-provided `unique_id`.
- Reuse existing local image content without redownload when possible.
- Materialize target file paths using hard links first, then copy fallback.
- Persist all tracked file paths into `image_locations`.
- Mark terminal status.

### Best-effort dedupe policy

Primary rule:

- dedupe by source-provided `unique_id`

Important limits:

- this is best effort only
- not all duplicate real-world images will be detected
- correctness must not depend on dedupe being perfect

Recommended lookup flow:

1. source returns image metadata
2. source implementation generates `unique_id`
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

- keep canonical images and per-device materialized paths under the same data directory when possible
- this maximizes chance that hard-linking works across supported deployments

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

For device wallpaper selection:

1. Wallpaper must come from a source the device subscribes to.
2. If device `is_adult_allowed = false`, wallpaper with `is_adult = true` must be excluded.
3. Wallpaper aspect ratio must fall within device tolerance.
4. Wallpaper resolution should be at least device size unless an optional fallback mode is introduced later.
5. Source and subscription must both be enabled.

Suggested SQL-level checks:

- `wallpaper.width >= device.screen_width`
- `wallpaper.height >= device.screen_height`
- `ABS(wallpaper.aspect_ratio - CAST(device.screen_width AS REAL) / device.screen_height) <= device.aspect_ratio_tolerance`

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

- some values may duplicate normalized wallpaper fields, but keeping them on `images` can simplify operational image listing
- alternatively, image list queries can join `wallpapers`; choose whichever keeps query complexity manageable

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

### Device endpoints

- `GET /api/devices`
- `POST /api/devices`
- `GET /api/devices/{id}`
- `PATCH /api/devices/{id}`
- `DELETE /api/devices/{id}`

### Source implementation metadata endpoints

- `GET /api/source-types`
  - list registered source implementation types from codebase
- `GET /api/source-types/{type}`
  - show validation/schema metadata for one source type, with params exposed as JSON Schema via `*huma.Schema`

### Configured source endpoints

- `GET /api/sources`
- `POST /api/sources`
- `GET /api/sources/{id}`
- `PATCH /api/sources/{id}`
- `DELETE /api/sources/{id}`

### Source schedule endpoints

- `GET /api/sources/{id}/schedules`
- `POST /api/sources/{id}/schedules`
- `PATCH /api/sources/{id}/schedules/{scheduleId}`
- `DELETE /api/sources/{id}/schedules/{scheduleId}`

### Subscription endpoints

- `GET /api/devices/{id}/subscriptions`
- `POST /api/devices/{id}/subscriptions`
- `PATCH /api/devices/{id}/subscriptions/{subscriptionId}`
- `DELETE /api/devices/{id}/subscriptions/{subscriptionId}`

### Wallpaper endpoints

- `GET /api/devices/{id}/wallpapers`
- `GET /api/wallpapers/{id}`

### Image endpoints

- `GET /api/images`
  - supports filters for adult flag, favorite, file size, width, height, uploader, artist, origin URL, and original ID
- `GET /api/images/{id}`
- `PATCH /api/images/{id}/favorite`
  - mark or unmark favorite
- `DELETE /api/images/{id}`
  - best-effort delete all tracked disk locations and then remove the image record

### Job endpoints

- `GET /api/jobs`
- `GET /api/jobs/{id}`
- `POST /api/sources/{id}/sync`
  - create manual source sync job and enqueue it

### Admin/runtime endpoints

- `POST /api/admin/scheduler/reload`
  - optional debug/admin endpoint
- `GET /api/admin/runtime`
  - optional runtime status including queue size, scheduler state, workers

### OpenAPI strategy

- Huma owns request/response types and OpenAPI generation.
- Frontend client is generated from the emitted OpenAPI document.
- DTOs stay separate from database models.
- Database field documentation should be descriptive enough that generated model metadata can later enrich API docs where appropriate.

## Go-Jet Customization Plan

Requirements:

1. Integer columns with prefix `is_` map to custom boolean type.
2. Columns with prefix `json_` and db type `json`, `jsonb`, or `string/text` map to custom `json.RawMessage` wrapper.
3. Column comments generate `dog:"${comment}"` tag.
4. Generated models also include `json:"${column_name}"` tag.

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

Important note:

- SQLite does not natively expose rich column comments like PostgreSQL/MySQL.
- If `dog` tags must come from column comments, Walens likely needs a sidecar schema metadata file or migration annotations consumed by the codegen wrapper.
- This remains one of the highest-risk implementation spikes.

### Schema documentation requirement

Every migration-defined field should carry documentation metadata.

Expectation:

- each table column should have a descriptive comment
- comments should explain the business meaning of the field, not just restate the name
- this documentation should flow into generated Jet model tags via `dog:"..."`
- that generated metadata should be usable later to enrich OpenAPI-facing DTO documentation

Practical implication for SQLite:

- since native column comments are weak or unavailable, Walens should plan for a schema metadata sidecar or migration annotation format as the source of truth for field documentation

## Database and Migration Plan

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

- Device form captures dimensions, adult toggle, and aspect tolerance.
- Source form selects source implementation type and edits params JSON.
- Source detail page manages many cron schedules.
- Job page shows status, run time, duration, counts, and error reason.
- Wallpaper grid shows preview, source, resolution, aspect ratio, and adult badge.
- Source param UI should be able to evolve from raw JSON editing into schema-driven forms using `*huma.Schema` / JSON Schema.

### Initial simplification

- Use validated JSON textarea for source params before building schema-driven forms.
- Show schedule proximity warnings as plain UI alerts first.

## Dev Server Integration Plan

Using `github.com/olivere/vite`:

- in development, Go server mounts or proxies Vite dev server assets
- in production, Go server serves built frontend assets from embedded files or packaged static directory
- API and frontend remain same-origin

Recommended route split:

- `/api/*` served by Huma
- non-API routes served by SPA fallback

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

- one config file or env vars
- one data directory for SQLite and future app-managed files

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

### Phase 1 - Runtime skeleton

1. Create app bootstrap, config, logging, database connection, and health endpoint.
2. Add migration runner.
3. Add runtime manager for HTTP server, scheduler, queue, and job runner in one process.
4. Add initial schema for devices, sources, source schedules, subscriptions, wallpapers, images, image_locations, and jobs.
5. Add Jet generation script/tooling.

### Phase 2 - Domain and API

1. Implement device repository/service/API.
2. Implement source registry and source-type metadata API.
3. Implement configured source repository/service/API.
4. Implement source schedule repository/service/API with cron validation and warning detection.
5. Implement device subscription API.

### Phase 3 - Jobs and Scheduling

1. Implement persisted jobs table and repository.
2. Implement in-memory queue.
3. Implement job runner with boot recovery.
4. Implement scheduler reload behavior on source/schedule mutations.
5. Add manual sync endpoint.

### Phase 4 - Ingestion and Wallpapers

1. Build normalized wallpaper model.
2. Implement one source first, ideally `BooruSource`.
3. Persist wallpaper metadata from jobs.
4. Implement image dedupe, local canonical storage, and image location tracking.
5. Implement per-device filtered wallpaper query endpoint.

### Phase 5 - Frontend SPA

1. Scaffold SvelteKit SPA.
2. Integrate generated API client.
3. Build devices UI.
4. Build source and schedule UI.
5. Build subscriptions UI.
6. Build jobs UI.
7. Build wallpaper gallery UI.
8. Build image management UI with search, filter, favorite, and delete flow.

### Phase 6 - Packaging

1. Embed/build production frontend assets.
2. Produce static Go binaries for all target platforms.
3. Produce single-container deployment.
4. Document local dev, binary deployment, and Docker deployment workflow.

## Risks and Open Questions

### 1. SQLite column comments

SQLite does not support column comments in the same rich way as PostgreSQL/MySQL.

Impact:

- `dog` tag generation cannot rely on normal SQLite schema reflection alone.

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

- store canonical downloaded files locally
- track all derived file paths in `image_locations`
- use hard links first and copy as fallback
- treat dedupe as optimization, not correctness guarantee

### 6. Cross-platform hard-link behavior

Hard-link support exists on target OSes but can vary by filesystem and mount layout.

Recommendation:

- explicitly test hard-link create/delete behavior on Windows, Linux, macOS, and Docker bind mounts
- keep copy fallback as a first-class supported path

## Recommended First Implementation Order

If implementation starts now, the best order is:

1. bootstrap one-process runtime with Go server + SQLite + migrations
2. validate pure-Go dependency set and cross-platform build matrix
3. validate Jet customization feasibility
4. implement sources + schedules schema/API
5. implement scheduler reload and warning detection
6. implement jobs persistence + in-memory queue + runner
7. implement images + image_locations schema and dedupe storage flow
8. implement device and subscription schema/API
9. implement wallpaper and image filtering query APIs
10. scaffold SPA and connect typed client
11. add first concrete source ingestion

## Deliverables for the Next Step

After planning, the next concrete implementation package should ideally produce:

- runnable one-process Go server skeleton
- migration set for devices, sources, schedules, subscriptions, wallpapers, images, image_locations, and jobs
- Jet generation script/prototype
- scheduler/queue/runner skeleton wired into app lifecycle
- Huma OpenAPI output
- SvelteKit SPA scaffold integrated with Vite mounting
- first CRUD endpoints for devices, sources, schedules, subscriptions, and image listing/favorite basics
