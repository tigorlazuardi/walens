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

## Plane MCP Task Rules

When working on a task that comes from Plane MCP:

1. mark the task as `in-progress` before starting implementation work
2. if there is ambiguity, missing scope, or meaningful implementation doubt, confirm with the user before implementing; otherwise implement directly without waiting
3. after the work is complete, create a git commit and push it, then provide a Markdown summary of what was done
4. add a comment to the Plane task containing the GitHub commit URL, and include a rich-text HTML version of the Markdown summary in that same comment
5. mark the task as `done` after the commit is pushed and the Plane comment has been added

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

Iterator contract rules:

- yield metadata lazily instead of buffering the full result set first
- honor `context.Context` cancellation promptly
- allow callers to stop iteration early without requiring full upstream drain
- surface iteration errors as they happen

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

Image materialization rules:

- canonical and device-facing stored images should be normalized to JPEG or PNG only
- if an image has transparency, materialize it as PNG
- if an image does not have transparency, materialize it as JPEG
- WebP and other supported input formats may be decoded for ingestion, but should not remain as device-facing stored output
- animated images should be rejected
- thumbnails are always JPEG, preserve aspect ratio, fit within `512x512`, and target about `40KB` on a best-effort basis

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
- doc metadata should become `doc:"..."` via generator-managed per-column metadata, not SQLite comment reflection

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
Do not rely on SQLite native comment reflection alone; preserve comment metadata in generator-managed per-column metadata that codegen can consume.

## Frontend Rules

Frontend is mobile first.

Architecture rules:

- use plain Svelte + Vite, not SvelteKit, unless the main plan is updated again
- backend owns the SPA HTML shell and injects runtime config through `window.__WALENS__`
- frontend routing must honor runtime `basePath` and must not assume deployment at `/`
- route modules should be lazy-loaded; prefer dynamic imports / `import.meta.glob`
- frontend API calls must derive from runtime `apiBase`, not hardcoded root-absolute URLs

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

### WALENS-2 / P0.2 - Vite mounting under subpath deployment

Validated on 2026-04-06 against Vite backend integration docs, Vite shared config docs, Svelte SPA deployment constraints, and `olivere/vite` documentation.

Outcome:

- `olivere/vite` is viable for Walens in both dev and production, but it only solves Vite asset integration; backend route ownership still needs to stay in Go
- build-time subpath configuration alone is not sufficient for a universal frontend artifact that must work at both `/` and `/walens`
- plain Svelte + Vite is preferred over SvelteKit because Walens needs runtime-derived base-path handling
- SPA fallback can coexist with login, docs, OpenAPI, and API routes as long as the frontend catch-all handler is registered last
- route-level code splitting remains viable as long as the frontend build emits relocatable asset URLs

Recommended implementation shape:

- keep the frontend mounted under the same backend base path chosen in runtime config
- use plain Svelte + Vite with a backend-owned shell that injects `window.__WALENS__` with at least `basePath` and `apiBase`
- use `olivere/vite` only for SPA shell + asset integration; keep login, docs, OpenAPI, and RPC/API handlers as normal backend routes
- use Vite production builds with `base: './'`, `manifest: true`, and preferably `publicDir: false` so emitted assets and split chunks stay relocatable under subpaths
- use a lightweight client router or custom router that derives its basename from runtime config
- lazy-load route modules via dynamic imports / `import.meta.glob`
- register SPA fallback last on the mounted app mux so `/login`, `/docs`, `/openapi.*`, and `/api/...` win before the client app catch-all
- in dev mode, keep the Vite dev server running separately and point `olivere/vite` at it via `ViteURL`
- in production, serve built assets from the generated frontend output and use the SPA fallback page for non-asset frontend routes
- optional background route preloading after first render is allowed

Why this shape was chosen:

- it preserves backend-controlled infra routes while still allowing SPA navigation under the same base path
- it allows one built frontend artifact to work at root and subpath deployments
- it matches Vite's documented backend integration model while avoiding compile-time-only frontend base-path assumptions

Watch-outs:

- frontend code must not hardcode root-absolute API or asset URLs
- split chunks and preloaded assets must be tested at both `/` and `/walens`
- if Vite `public` directory is used, its assets must be served consistently in both dev and prod; disabling `publicDir` is often simpler for backend-driven apps
- dynamic imports must resolve relative to the served entry asset location, not a root-only assumption
- in dev mode, asset URL handling may require Vite proxying or `server.origin` depending on how backend HTML references are emitted

Plan updated to move the frontend architecture from SvelteKit SPA to plain Svelte + Vite with runtime-injected base path handling.

### WALENS-3 / P0.3 - SQLite stack and migration approach

Validated on 2026-04-06 against the no-CGO runtime constraints and planned Go-Jet customization workflow.

Outcome:

- `modernc.org/sqlite` remains aligned with Walens pure-Go deployment goals
- `golang-migrate` remains aligned with embedded startup migrations
- migration 1 should remain SQLite setup and optimization only
- SQLite comment support is not sufficient as a source for Jet `doc` tags
- `doc` tags should be emitted by a generator hook at table/column granularity instead

Recommended implementation shape:

- keep SQL migrations authoritative for schema only
- keep field documentation metadata in the codegen layer and update it alongside schema changes
- use a helper shaped like `func createDocTag(table metadata.Table, column metadata.Column) (tag string)` returning `doc:"..."`
- fail generation when required documentation metadata is missing for business schema columns

Why this shape was chosen:

- it avoids relying on unsupported SQLite comment reflection
- it fits the existing need for Go-Jet customization
- it keeps generated docs explicit and deterministic

Watch-outs:

- migration changes and doc metadata changes can drift if not updated together
- column renames need matching generator metadata updates
- generator validation should fail loudly on missing documentation

### WALENS-4 / P0.4 - Go-Jet customization pipeline

Validated on 2026-04-06 against the required mappings for UUID IDs, Unix-millisecond timestamps, duration wrappers, doc tags, identifier naming, and Huma schema integration.

Outcome:

- the Go-Jet customization path is viable through a Walens-owned generation wrapper
- generated files should not be hand-edited; deterministic rewrite/generation steps should own all customizations
- `id` and `*_id` UUID mapping remains viable, but only if external IDs consistently use `*_identifier`
- timestamp wrapper mapping is viable, but should use explicit field matching instead of broad integer heuristics
- duration wrapper mapping is viable for true `*_ms` duration columns
- Huma `SchemaProvider` support should live on shared wrapper types used in API-facing models

Recommended implementation shape:

- run migrations into a local codegen SQLite database
- run Jet generation against that schema
- run a Walens post-generation step to rewrite generated field types and tags deterministically
- centralize helpers such as `isUUIDColumn`, `isUnixMilliTimestampColumn`, `isDurationColumn`, and `applyDocTag`
- require wrapper types used by generated DB models to implement `sql.Scanner` and `driver.Valuer`

Watch-outs:

- broad suffix-based rules can rewrite the wrong columns
- naming drift from `*_identifier` back to `*_id` will break the intended internal/external ID split
- broken wrapper DB conversion logic will surface as runtime query/scan failures

### WALENS-5 / P0.5 - Pure-Go thumbnail and filesystem behavior

Validated on 2026-04-06 against the wallpaper compatibility goals for Linux, Windows, macOS, and mobile-target devices.

Outcome:

- filesystem materialization should stay on the Go standard library
- canonical and device-facing outputs should be normalized to JPEG or PNG only
- images with transparency should materialize as PNG
- images without transparency should materialize as JPEG
- WebP may be accepted as input but should be converted before storage
- animated images should be rejected
- thumbnails should always be JPEG, fit within `512x512`, preserve aspect ratio, and target about `40KB` best-effort

Recommended implementation shape:

- use standard library file operations for temp download, rename, hard-link creation, copy fallback, and cleanup
- use a pure-Go scaler from `golang.org/x/image` for thumbnail generation
- use standard library JPEG and PNG encoders for final stored output
- treat the thumbnail size target as advisory rather than a hard failure condition

Watch-outs:

- transparency detection must be correct because it decides output format
- animated image rejection should happen before partial materialization
- some thumbnails will remain above target size and should still be accepted after best effort

### WALENS-6 / P0.6 - Source iterator fetch contract

Validated on 2026-04-06 against the requirement that fetched source metadata should stream progressively into downstream processing.

Outcome:

- the iterator-based `Fetch(ctx, req) iter.Seq2[ImageMetadata, error]` contract is viable
- source fetching should stay lazy so downstream work can begin before the full source scan completes
- callers should be able to stop iteration early without draining the full upstream result set
- `context.Context` cancellation should be honored promptly
- iteration errors should surface as they happen rather than at the end of the scan

Recommended implementation shape:

- keep `Fetch` iterator-based in the source interface
- yield metadata items as upstream responses are decoded or discovered
- thread `context.Context` through upstream HTTP and decode loops
- avoid full-result buffering unless the upstream API forces it

Watch-outs:

- some upstream APIs may still require page-level buffering
- cancellation checks must exist inside pagination and decode loops
- iterator error delivery should stay deterministic so partial progress is understandable

## Editing Discipline

When implementing:

- keep naming aligned with the plan
- do not reintroduce a separate `wallpapers` persistence phase
- do not move auth settings into persisted DB config
- do not add root-only routing assumptions
- do not introduce non-pure-Go backend dependencies
- do not add user/account systems

If a change alters architecture, update `plans/initial-architecture-plan.md` in the same session.
