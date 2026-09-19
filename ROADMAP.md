# GitPatrol Roadmap

This document outlines the development milestones for **GitPatrol**, an autonomous repository monitoring and archival suite.

## Roadmap

1. [x] **Multi-provider abstraction (GitHub, GitLab, etc.)**
   - Introduced a `Source` interface in `source.go`.
   - Used a factory pattern (`GetSource`) to select the source by repository URL.

2. [x] **Data extraction logic for non-git metadata (Issues, Wikis, Release Notes)**
   - Issues, Wikis, and text-only Release Notes are now archived locally.
   - **UI Polish:** Implemented a Tab-based interface (Briefing, Intel, Chronicle, Wiki, Logs) to view this data.

3. [x] **UX & Stability Polish**
   - **Manual Sync:** Added "Sync Now" functionality for on-demand updates.
   - **Human-Readable Intervals:** Support for `1h 30m` style configuration.
   - **Advanced Error Reporting:** Replaced browser alerts with a premium **Toast system** and implemented human-readable Git CLI error messages.
   - **Scheduler Fix:** Optimized the sync engine to respect `auto_patrol` and avoid redundant scans.

4. [x] **URL Normalizer (UX Improvement)**
   - Automatically convert SSH URLs to HTTPS.
   - Strip sub-paths (e.g., `/tree/main/...`) from browser-pasted URLs.
   - Support shorthands like `user/repo` for GitHub.
   - **UI Polish:** Added real-time URL normalization preview and automatic Name suggestion in the dashboard.

5. [x] **Reliable Sync Engine (Standardized on Git CLI)**
   - Decided to stick with the official Git CLI for 100% branch/tag parity.
   - Optimized with `--mirror` clones and SQLite WAL-mode for maximum concurrency and reliability.

6. [x] **Git-native Metadata Reading**
   - Switched from reading files (README) from disk to reading them directly from the Git database (e.g., `git show origin/main:README.md`).
   - **Feature:** Added a generic asset handler that serves images and other files directly from the Git object store, ensuring the dashboard is always up-to-date with the latest `fetch` without needing a merge.

7. [x] **Incident Reporting & Monitoring (Persistent Logging)**
   - Created an `incidents` table to store historical sync failures.
   - **UI:** Added a "Security Logs" modal triggered by a notification bell to review errors.
   - **Feature:** Implemented deduplication and permanent failure detection (auto-disabling patrol).
   - **Health Badges:** Implemented dynamic SVG badges per repository.

8. [x] **Concurrency & Rate Limiting**
   - **Worker Pool:** Implemented a limited set of workers to handle sync tasks sequentially and prevent CPU/IO spikes.
   - **Unique Queue:** Ensured a repository cannot be queued multiple times.
   - **Adaptive Sync:** If an API rate limit is reached, metadata fetching is skipped but the core Git Sync continues.

9. [x] **Community Authentication (Single-User)**
   - **Logic:** If no user exists, show a "Bootstrap" page to create the initial account.
   - **Single User:** CE only supports one admin user for simplicity.
   - **Local-Friendly:** Use Http-Only cookies with `Secure: false` by default. Configurable via `GP_SECURE_COOKIE`.
   - **Middleware:** Protect all dashboard API routes.
   - **UX:** Implemented auto-logout on session expiration (401).

10. [x] **Comprehensive Documentation & Standardization**
    - Updated `README.md` and `docs/` to reflect current architecture and features.
    - Standardized error handling and API communication.
    - Added instructions for Health Badge embedding.

11. [x] **One-click Repository Export (Manager of Choice)**
    - Implement a bridge to push local mirrors directly to a destination Git host (e.g., Gitea, Forgejo, GitHub, GitLab).
    - **Logic:** Added a "Deploy to..." button in the repository detail view.
    - **UX:** One-click transfer of archived code and metadata to a fresh instance for recovery or migration.

12. [x] **Provider Token Support via Environment Variables**
    - **Logic:** Allow users to set global tokens (e.g., `GITHUB_TOKEN`, `GITLAB_TOKEN`) via `.env`.
    - **Feature:** Automatically inject these tokens into sync requests to avoid rate limits and allow cloning of private repositories (within the user's scope).
    - **Security:** Ensure tokens are only used for the intended provider and never logged.

13. [x] **Docker Hub Support**
    - Provide an official Docker image on Docker Hub for rapid deployment.
    - **Optimization:** Multi-stage build producing a minimal Alpine-based image with the single binary.
    - **Automation:** GitHub Action to build and push images on every release tag.

14. [x] **Proxmox LXC Install Script & Systemd Persistence**
    - Create a dedicated install script for Proxmox (LXC container).
    - **Persistence:** Provided `systemd` unit files to ensure GitPatrol starts automatically on boot and recovers from crashes.
    - **Release:** Automated GitHub Release workflow for binary distribution.

15. [x] **Surgical Health & Status Monitoring**
    - **Backend:** Implemented a `/api/health` endpoint monitoring Internet connectivity (8.8.8.8), Disk Space availability, and Database integrity.
    - **Autonomy:** Backend logs system incidents and broadcasts health status via WebSockets.
    - **Frontend:** Upgraded the "System Online" badge to a dynamic traffic-light system with real-time metrics.

16. [ ] **Production Hardening & Storage Persistence**
    - Configurable `DATA_DIR` and `DB_PATH` to ensure Docker volumes (`/var/lib/gitpatrol`) persist across container restarts.
    - ID-prefixed repository storage (`data/repos/{id}_{name}`) with auto-migration of legacy paths.
    - Dynamic `PORT` binding and graceful process shutdown (`SIGINT`/`SIGTERM`) handling.

17. [ ] **Authentication & Security Hardening**
    - Enable operator password changing in user profile with current password verification.
    - Secure JWT verification in `CheckAuthStatus` and enforce `SameSite` & `GP_SECURE_COOKIE` flags.
    - Rate limiting on `/api/auth/login` and `/api/auth/register`.

18. [ ] **Git Mirror Engine & Private Repository Support**
    - Ingest `GITHUB_TOKEN` and `GITLAB_TOKEN` into Git CLI operations for private repositories.
    - Command context timeouts (anti-hang protection) during Git synchronization.

19. [ ] **Offline Recovery (.ZIP Downloads) & Incident Telemetry**
    - Implement `GET /api/repositories/:id/download` for one-click `.zip` archive recovery.
    - Implement `POST /api/incidents/:id/resolve` for incident acknowledgement.
    - Compute real fleet storage usage and uptime on dashboard hero banner.

20. [ ] **Test Suites & Quality Assurance**
    - Add automated Go unit tests for auth, health calculation, and URL normalization.
