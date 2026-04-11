# Walens

Walens is a small self-hosted wallpaper collection app. You define where to fetch images from (sources), which devices you want to wallpaper, and when to run collection jobs. Walens handles the rest — fetching, filtering, downloading, storing, and tracking images for each device.

The web UI lets you manage sources, devices, schedules, and browse your collected images. Everything runs in a single process with no external services required.

---

## Who Walens Is For

Walens is designed for self-hosters who want to automate wallpaper collection across one or more devices without relying on third-party services or cloud platforms.

Typical use cases:

- You have multiple computers or TVs that each need wallpapers, and you want a centralized way to manage where images come from and how they are filtered.
- You want to pull images from tag-based image sources (like booru-style boards) and automatically filter them by resolution, aspect ratio, and file size before they reach your devices.
- You want a simple, single-deploy solution that does not require setting up databases, queues, worker processes, or user accounts.

Walens is **not** a user platform, a collaborative system, or a content management product. It does not have user accounts, roles, or permissions beyond a single optional password.

---

## Core Workflow

### Step 1 — Add a Source Type

A *source type* is a built-in image provider built into the Walens code (for example, a booru-style tag-based source). You do not install source types — they are part of the application. You can see available source types through the API or UI.

### Step 2 — Create a Source

A *source* is a configured instance of a source type. When you create a source, you give it a name, pick a source type, and fill in the type-specific parameters (such as which tags to search for).

Example: you might create a source named "landscape-pics" using the `booru` source type with params `{"tags": ["landscape"]}`.

Each source also has a **lookup count** — this controls how many upstream items Walens checks per run (not how many images it returns). A higher count means more filtering work but more chances to find matches.

### Step 3 — Register Your Devices

A *device* represents a wallpaper target — a monitor, TV, phone, or any screen. For each device you configure:

- Screen resolution (width × height)
- Aspect ratio tolerance — how much the wallpaper aspect ratio can deviate from your screen's ratio
- Optional dimension bounds (minimum/maximum image width and height)
- Optional file size bounds
- Adult content setting (allow or block adult images)

### Step 4 — Subscribe Devices to Sources

A *device source subscription* links a device to a source. Only subscribed devices receive images from a source when it runs. This is how you control which images go where.

One device can subscribe to many sources. One source can be subscribed to by many devices.

### Step 5 — Set a Schedule

A *source schedule* binds a cron expression to a source. When the schedule fires, Walens creates a background job that:

1. Checks whether the source is enabled.
2. Loads the subscribed devices for that source.
3. Fetches image metadata from the upstream source.
4. Filters each image against every subscribed device's constraints.
5. Downloads matching images, materializes them for each eligible device, and generates thumbnails.

### Step 6 — Browse Images and Job History

In the UI you can:

- Browse all collected images, filter by device, favorite status, dimensions, file size, and more.
- View job history — see how many images were downloaded, reused, hard-linked, or skipped per run and why.
- Manually trigger a source sync outside its schedule.
- Mark images as favorites or blacklist them to prevent future downloads.

---

## Main Concepts

### Source Types vs. Sources

A **source type** is a built-in image provider registered in the application code (for example, `booru`). It defines how to talk to an upstream service and what parameters it accepts.

A **source** is a user-created configuration row. It references one source type and stores the parameter values for your specific use case.

Think of it like this: the source type is the "driver," and the source is the "configured connection."

### Devices

A device is a physical or logical wallpaper target. Each device has its own screen constraints and its own folder of stored images on disk. The same image can be stored for many devices simultaneously.

### Source Schedules

A source schedule is a cron trigger attached to a source. When the schedule fires, a background job runs. One source can have multiple schedules (for example, different times of day).

### Device Source Subscriptions

A subscription connects a device to a source. When that source runs, only devices with an active subscription to it are candidates to receive images. Disabling a subscription pauses that source's delivery to that device without deleting the subscription or the source.

### Images

An image record is created when Walens successfully processes an image from a source. It stores metadata such as dimensions, file size, artist, uploader, origin URL, and tags. The record persists even if the physical file is later removed from disk.

### Favorites

You can mark any image as a favorite. Favorites survive source re-runs and are not automatically deleted. Use favorites to flag images you want to keep regardless of future source updates.

### Blacklist

The blacklist prevents a specific image from being downloaded again. It is keyed by source + unique image identifier. Blacklisted images are skipped on future runs even if they match device filters.

### Jobs

A job is a record of a source run. Jobs track what happened: how many images were fetched, how many were downloaded fresh, how many were reused from existing local files, how many were hard-linked or copied to additional devices, and any errors that occurred.

---

## Key Features

- **Scheduled wallpaper collection** — set cron schedules per source; jobs run automatically in the background.
- **Device-based filtering** — images are filtered by resolution, aspect ratio, file size, dimension bounds, and adult content settings before download.
- **Smart deduplication** — Walens reuses existing local image files instead of re-downloading when possible.
- **Hard links and copy fallback** — when the same image is needed on multiple devices on the same filesystem, Walens uses hard links to save space; copy is used when hard links are not possible.
- **Thumbnail generation** — every downloaded image gets a UI-optimized JPEG thumbnail automatically.
- **Device-oriented storage** — each device has its own folder of images, making it easy to use with `rsync` or `syncthing` to sync to the actual device.
- **Image favorites and blacklist** — manually flag images to keep or exclude.
- **Job history** — full visibility into what each source run did.
- **Optional Basic Auth** — protect your self-hosted deployment with a single username and password.
- **Single binary deployment** — no external database, no worker processes, no queue broker.

---

## Deployment Model

Walens is built for self-hosters who value operational simplicity.

### What You Get

- **One static binary** for your target OS and architecture (Windows, Linux, macOS on amd64 or arm64).
- **One Docker image** for container deployments.
- **One SQLite database** stored alongside (or in) your data directory — no separate database server needed.

### What You Do Not Need

- No separate worker binary.
- No external message queue or broker (RabbitMQ, Redis, etc.).
- No background job processing service.
- No multi-node setup or high-availability infrastructure.
- No compiled native dependencies.

### Runtime Characteristics

- **Single process** — HTTP server, scheduler, queue, and job runner all run in one OS process.
- **SQLite** — all data is stored in a single SQLite file using a pure-Go driver.
- **Cross-platform** — builds for Windows (amd64, arm64), Linux (amd64, arm64), and macOS (amd64, arm64).
- **No CGO, no native bindings** — the binary is fully static and self-contained.

This makes Walens easy to drop onto a home server, a small VPS, or run inside a simple Docker container.

---

## Authentication

Walens supports optional Basic Auth to protect self-hosted deployments.

- Auth is disabled by default. When disabled, anyone who can reach the URL can access the UI and API.
- When enabled, you set a username and password through environment variables or a config file. Credentials cannot be changed through the UI.
- Browser users get an HTTP-only cookie after logging in. Subsequent browser requests authenticate via cookie.
- API clients (scripts, mobile apps, other tools) should send the `Authorization` header directly.

This is intentionally simple. There are no user accounts, sessions, roles, or permission levels — just one optional password protecting the entire deployment.

---

## How Images Are Stored

### Device-Oriented Layout

Images are stored in folders organized by device:

```
{data_dir}/images/{device_slug}/{image_unique_identifier}.{ext}
```

For example, if you have a device named `living-room-tv`, its wallpapers live in `images/living-room-tv/`. This layout is designed for easy integration with `rsync` and `syncthing` — you can point those tools at a device folder to sync its wallpapers to the actual machine.

One image can exist in multiple device folders simultaneously. Walens tracks every location on disk.

### File Normalization

Downloaded images are converted to a standard format:

- **PNG** if the image has transparency
- **JPEG** if it does not

WebP and other input formats are accepted but are converted during materialization. Animated images are rejected rather than partially stored.

### Thumbnails

Every downloaded image gets a thumbnail generated for UI listing and preview. Thumbnails are:

- Always JPEG
- Fit within a 512×512 bounding box
- Aspect ratio is preserved
- Target around 40 KB in size

Thumbnails are generated after the canonical image file is available and are tracked in the database alongside the image record.

### Assignment and Materialization Rules

When a source runs and finds an image that matches a device:

1. **Already assigned + file exists** → skip, no work needed.
2. **Already assigned + file missing** → re-download or re-materialize for that device.
3. **Assigned to another device + file exists** → create a hard link if possible; copy as fallback.
4. **Assigned to another device + file missing** → download fresh for the current device only; do not change assignment state for other devices.

Blacklisted images are skipped before any download begins.

---

## Current Scope and Non-Goals

### What Walens Does

- Run wallpaper collection jobs on schedule
- Filter images by device constraints (resolution, aspect ratio, file size, adult content)
- Download and store images efficiently
- Generate thumbnails
- Track which images are assigned to which devices
- Expose a web UI for managing sources, devices, schedules, and browsing images

### What Walens Does Not Do (Now)

- User accounts, user management, or per-user settings
- Multi-user access control, roles, or permissions beyond a single deployment password
- Collaborative features, sharing, or public galleries
- High availability or multi-node deployments
- External queue brokers or distributed workers
- Advanced image recommendation or ranking beyond tag/dimension filtering
- Native mobile apps (a web UI is provided)

### Future Possibilities

These are explicitly out of scope for now but are listed so the current non-goals are clear:

- Mobile apps
- User management
- Advanced recommendation algorithms
- HA/multi-node setups

---

## API and UI

### Web UI

Walens ships a responsive web UI built with Svelte. The UI lets you manage all entities — sources, devices, schedules, subscriptions, images, and jobs — without needing API tools.

### API

The backend exposes a RPC-style JSON API. All methods are `POST` with JSON request bodies. Routes follow the pattern:

```
/api/v1/{domain}/{MethodName}
```

For example: `POST /api/v1/images/ListImages`, `POST /api/v1/devices/CreateDevice`.

API documentation is available at `/docs` (Scalar UI), `/openapi.json`, and `/openapi.yaml` when the server is running.

### Base Path Support

Walens can be deployed at the root path (`/`) or under a subpath (e.g., `/walens`). Configure this at startup via the `WALENS_BASE_PATH` environment variable. All routes, assets, and API endpoints honor the configured base path.

---

## Quick Reference

| Concept | Description |
|---|---|
| Source Type | Built-in image provider (e.g., booru). Not user-configurable. |
| Source | A configured instance of a source type with name and params. |
| Device | A wallpaper target with screen constraints. |
| Source Schedule | A cron trigger for a source. |
| Device Subscription | Links a device to a source — controls which devices receive from which sources. |
| Image | A persisted image record with metadata from the source. |
| Job | A record of a source run — counts, status, timing, and errors. |

---

## Status

Walens is under active development. The core workflow described in this document is the target state.
