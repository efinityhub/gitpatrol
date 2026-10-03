---
sidebar_position: 2
title: Architecture
---

# Architecture ⚡

GitPatrol follows a modern decoupled architecture designed for high performance and low-overhead repository mirroring.

## System Overview
The system consists of three main components:

1.  **Backend (Go):** A high-concurrency engine built with Go, responsible for:
    - Managing the SQLite database via `modernc.org/sqlite`.
    - Interfacing with the Git CLI for efficient mirroring using `--mirror` clones.
    - Exposing a REST API using the **Echo** framework.
    - Implementing a **Worker Pool** to limit concurrent synchronization tasks.
    - Securing routes with **JWT Middleware** and Http-Only cookies.
2.  **Frontend (SvelteKit):** A reactive dashboard powered by SvelteKit and Bun, providing:
    - A visual overview of all mirrored repositories.
    - Health score visualizations and activity sparklines.
    - Real-time status updates via WebSockets.
3.  **Data Persistence:**
    - **SQLite:** Stores configuration, health scores, and metadata.
    - **Local Disk:** Stores the physical Git database, avatars, and archived JSON/Markdown metadata (Issues, Wiki).

## Data Flow
1.  **Repository Added:** User submits a URL via the UI.
2.  **Initial Clone:** The backend performs a full mirror clone of the repository.
3.  **Metadata Extraction:** The backend pulls the latest stars, forks, issues, releases, and wiki pages.
4.  **Scheduled Sync:** Based on the user-defined interval, the backend fetches updates from the remote.
5.  **Health Score:** On each sync, the system recalculates the "Health Score" based on activity, stars, and remote availability.

## Security Model
- **Authentication:** Single-user Community Edition restricted to one admin user.
- **Session Management:** Secure Http-Only cookies with configurable `Secure` flags.
- **Password Safety:** Bcrypt hashing with a system-level pepper generated on first run.
- **CSRF Protection:** Session cookies are `SameSite=Lax`.
- **Brute-force Protection:** Login and registration share a per-client rate limit (5 attempts, then one every 12 seconds). Clients are identified by their direct connection address, so behind a reverse proxy all users share one budget.
- **Graceful Shutdown:** On `SIGINT`/`SIGTERM` GitPatrol stops accepting requests, lets running syncs finish for up to 30 seconds (then cancels them and returns the repository to `pending`), flushes queued logs, and closes the database.

## Design Principles
- **Minimalist:** No heavy frameworks where not needed (e.g., vanilla DOM manipulations in high-performance areas).
- **Offline-First:** All repo data is mirrored locally; the dashboard serves READMEs and assets directly from the local Git store.
- **Resilient:** Uses SQLite WAL-mode for safe concurrent access.
