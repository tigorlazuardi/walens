# AGENTS.md

This file is the fast re-entry guide for fresh agent sessions and post-compaction recovery.

The authoritative architecture document is `plans/initial-architecture-plan.md`.
If this file and the plan ever drift, update both or prefer the plan and fix this file.

## Session Start Rules

Before making architectural or implementation changes, re-read:

1. `AGENTS.md`
2. `plans/initial-architecture-plan.md`

Do not silently invent a conflicting architecture.
If a requested change conflicts with the plan, either:

- update the plan first, or
- explicitly call out the conflict to the user.

## Product Scope

Walens is a small self-hosted wallpaper collection app.

Core purpose only:

- run wallpaper collection jobs on schedule
- filter images that match device constraints
- download and store images efficiently
- expose a small self-hosted web UI for managing that flow

Do not expand it into:

- user management
- multi-tenant access control
- collaboration features
- HA/distributed worker architecture
- heavy workflow/business logic beyond the wallpaper pipeline

Basic Auth is deployment protection only, not a user/account system.

## Non-Negotiable Runtime Constraints

- single process only
- HTTP server, scheduler, queue, and runner live in the same process
- no HA assumptions
- no distributed locks or external queue
- no CGO
- no native bindings
- no FFI
- SQLite must use a pure-Go driver

Target outputs:

- static binaries for Windows, Linux, macOS
- both `amd64` and `arm64`
- one Docker image

## Dependency Rules

Backend:

- pure Go only
- no CGO-only libraries
- no native shared-library requirements

Frontend:

- avoid native/runtime-bound dependencies where reasonably possible
- prefer portable JS tooling

## High-Level Model

Important persisted entities:

- `devices`
- `sources`
- `source_schedules`
- `device_source_subscriptions`
- `images`
- `tags`
- `image_tags`
- `image_assignments`
- `image_locations`
- `image_thumbnails`
- `image_blacklists`
- `jobs`
- `configs`

There is no separate discovered-wallpaper phase.
Fetched source metadata is persisted directly into `images`.

## Source Model Rules

Source implementations live in code and have unique names, for example `booru`.

The `sources` table stores user-created configured source rows.

Each source row has:

- `name`
- `source_type`
- `params`
- `lookup_count`
- schedules

Devices subscribe to source rows, not directly to implementation types.

`lookup_count` means upstream lookup budget, for example "check the latest 300 posts".
It is not a guarantee of returned image count.
Skipped, non-image, and deduped items still count toward lookup budget.

Use this source contract shape:

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
```

`Fetch` is iterator-based and should support progressive downstream work.

## Background Runner Rules

Required source-run flow:

1. if source is disabled, exit immediately and do not create a job log
2. when cron triggers, load enabled subscribed device candidates
3. if none exist, create a job log with an English informational message and stop
4. fetch image metadata
5. compare each image to candidate devices
6. if no device matches, skip that image
7. if one or more devices match, pass the image plus eligible devices downstream

Assignment/materialization rules:

1. assigned to device and file exists -> skip
2. assigned to device and file missing -> redownload/materialize for that device
3. assigned to another device, not this one -> hard link first, copy fallback
4. assigned elsewhere but source file missing -> download for current device only; do not implicitly reassign other devices

Blacklist rule:

- if `(source_id, unique_identifier)` exists in `image_blacklists`, never download/redownload it

Download rule:

- always download to temp location first
- only move/copy into final device path after success
- generate thumbnail after the canonical device file is available

## Storage Rules

Filesystem layout must stay device-oriented:

```text
{base_dir}/images/{device_slug}/{image_unique_identifier}.{ext}
```

Why:

- easy integration with `rsync` and `syncthing`
- one image can exist in many device-specific locations
- therefore `image_locations` tracking is required

Prefer:

- no redownload if local image already exists
- hard link before copy
- copy only as fallback

Thumbnails are mandatory post-download artifacts for UI performance.
They must use a pure-Go implementation.

## Config Rules

Application config:

- single source of truth is the `configs` table
- one row in practice
- full JSON value replacement only
- insert defaults at boot if empty
- no layered config merge system

Auth config is separate:

- bootstrap/runtime only
- not stored in `configs`
- not editable in UI

## Auth Rules

Optional Basic Auth only.

When enabled:

1. check `Authorization` header first
2. if missing, check HTTP-only cookie
3. invalid or missing -> `401`

Browser users get a simple login page.
Future mobile/external clients should use `Authorization` header.

No users table. No account model. No role system.

## Routing Rules

Base path is configurable.
Default is `/`, but app may be mounted under subpath like `/walens`.

Everything must honor base path:

- API routes
- login page
- docs
- OpenAPI endpoints
- SPA fallback
- Vite dev assets
- production assets

API prefix is always `{base}/api`.

RPC route convention:

```text
{prefix}/{version}/{domain}/{MethodName}
```

Example:

```text
POST /api/v1/images/ListImages
POST /walens/api/v1/images/ListImages
```

Normal backend methods:

- use `POST`
- use JSON body

Exceptions allowed for infra routes only, such as:

- `GET {base}/login`
- `GET {base}/docs`
- `GET {base}/openapi.json`
- `GET {base}/openapi.yaml`
- static assets
- health/docs delivery routes

## Backend File Structure Rules

Use this structure:

```text
internal/routes/register.go
internal/routes/[domain]/{MethodName}.go
internal/services/[domain]/[domain].go
internal/services/[domain]/{MethodName}.go
```

Route layer rules:

- thin only
- define Huma operation metadata
- translate DTOs if needed
- call service methods
- do not hold business logic

Service layer rules:

- own business logic
- own DB work
- own orchestration across scheduler/queue/runner/storage

## Data/Codegen Rules

Go-Jet generation rules must remain consistent:

- `is_*` integer columns -> custom boolean type
- `json_*` text/json-like columns -> custom raw JSON type
- `id` and `*_id` -> `uuid.UUID`
- internal IDs are UUIDv7
- external/source-owned IDs must use `*_identifier`, not `*_id`
- timestamp INTEGER milliseconds -> custom time wrapper
- duration INTEGER milliseconds via `*_ms` duration fields -> custom duration wrapper
- generated fields also include `json` tags
- doc metadata from migration comments should become `doc:"..."`

Custom types exposed through API models must implement Huma `SchemaProvider`.

Important transport/storage split:

- timestamps in DB: INTEGER Unix milliseconds
- timestamps in JSON/OpenAPI: same shape as `time.Time` / RFC3339Nano
- durations in DB: INTEGER milliseconds
- durations in JSON: integer milliseconds; input may accept string or integer milliseconds

## Migration Rules

Migration order is strict:

- migration 1: SQLite optimizations and best practices only
- migration 2+: business schema

Every schema field should have descriptive documentation metadata.
Do not rely on SQLite native comment reflection alone; preserve comment metadata in a way codegen can consume.

## Frontend Rules

Frontend is mobile first.

Required UX rules:

- smooth on mobile first
- image listings use masonry layout
- minimum 2 columns on phone-sized view
- dynamically increase columns on wider viewports
- prefer thumbnails over original files in list/gallery views

Performance rules:

- aggressively lazy-load heavy components
- aggressively lazy-load dependency-heavy views
- avoid eager rendering of off-screen/heavy image UI where possible

## Phase 0 Validation Log

### WALENS-1 / P0.1 - Huma runtime and configurable base path

Validated on 2026-04-06 against Huma v2 documentation.

Outcome:

- Huma is viable on top of `net/http` via the `humago` adapter for Go 1.22+ `http.ServeMux`
- Configurable base-path deployments are viable for both `/` and subpaths like `/walens`
- Huma docs/OpenAPI endpoints can stay at `{base}/docs` and `{base}/openapi.{json,yaml}` while RPC methods live under `{base}/api/...`
- Plain `net/http` handlers can coexist with Huma routes for login and other infra endpoints

Recommended implementation shape:

- use one root `http.ServeMux` for the process
- create one app sub-mux and mount it at the configured base path; if base is `/`, use it directly
- create the Huma API on that app sub-mux via `humago`
- register RPC operations with explicit paths starting at `/api/v1/...` instead of mounting Huma itself under `/api`
- keep login and similar infra pages as plain `net/http` handlers on the same mounted app mux
- keep auth enforcement as outer HTTP middleware with allowlist support for `GET {base}/login`; prefer header first, then cookie fallback

Why this shape was chosen:

- it preserves the required route contract without needing separate routers for docs vs API
- it keeps base-path handling centralized at the mux mount point
- it stays aligned with the single-process `net/http` architecture in the main plan

Watch-outs:

- `humago` requires Go 1.22+ because it targets the newer `http.ServeMux`
- if `config.Servers` is set for OpenAPI generation, ensure the configured URL/path includes the deployed base path so docs generate correct client URLs
- avoid mounting Huma under `/api` if docs and OpenAPI must remain at `{base}/docs` and `{base}/openapi.*`
- auth middleware must not block the login page itself

No plan/task changes required from this validation result.

## Editing Discipline

When implementing:

- keep naming aligned with the plan
- do not reintroduce a separate `wallpapers` persistence phase
- do not move auth settings into persisted DB config
- do not add root-only routing assumptions
- do not introduce non-pure-Go backend dependencies
- do not add user/account systems

If a change alters architecture, update `plans/initial-architecture-plan.md` in the same session.
