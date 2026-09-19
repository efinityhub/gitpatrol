---
sidebar_position: 1
title: Features
---

# Core Features 🚀

GitPatrol provides a suite of tools for repository mirroring and monitoring, designed for high-concurrency and reliability.

## 📦 Professional Mirroring
- **Accumulative Fetching:** Uses `git fetch --all --tags --force` to ensure all remote objects are pulled locally.
- **Auto-Mirroring:** Periodically polls repositories for changes based on configurable intervals.
- **Avatar Archiving:** Automatically downloads and mirrors provider avatars (e.g., GitHub profile pictures) for offline display.
- **Asset Service:** Serves images and other files directly from the Git object store, ensuring the dashboard is always up-to-date with the latest `fetch`.

## 📊 Health Monitoring & Intel
The "Health Score" is a unique metric calculated on each sync to help you assess the vitality of a project:
- **Activity Check:** Analyzes commit frequency over the last 14 days.
- **Community Stats:** Tracks Stars, Forks, and Open Issues via provider APIs.
- **Metadata Archiving:** Locally archives Issues, Releases, and Wikis for offline reference.
- **Staleness Analysis:** Decreases score if the last commit was more than 30, 90, or 365 days ago.
- **Remote Check:** Verifies if the repository still exists at its remote URL.

## 🖥️ Modern Dashboard
- **Real-Time Updates:** Uses WebSockets to stream sync progress and logs directly to the browser.
- **Tabbed Interface:** Switch between **Briefing** (README), **Intel** (Issues), **Chronicle** (Releases), **Wiki**, and **Logs** (Commits).
- **Commit History Visualization:** See a 14-day activity sparkline for every repository at a glance.
- **Status Indicators:** Clear visual cues for `pending`, `syncing`, `synced`, or `error` states.

## 🏷️ Health Badges
GitPatrol can generate dynamic SVG badges for any repository you are monitoring. These badges are perfect for embedding in your repository READMEs or external status pages.

### Usage
You can access a repository's badge via the following URL pattern:
`http://your-gitpatrol-instance:8080/api/repositories/{id}/badge`

### Embedding in Markdown
```markdown
![GitPatrol Health](http://your-gitpatrol-instance:8080/api/repositories/1/badge)
```

## 🔐 Security & Monitoring
- **Single-User CE Auth:** Community Edition supports a secure, single-user administrative login.
- **Bootstrap Mode:** Automatically detects if no user exists and prompts for system initialization.
- **Incident Logs:** Dedicated security logs and notification system for tracking synchronization failures.
- **Worker Pool:** Intelligent task queueing to prevent CPU/IO spikes during heavy mirroring tasks.
- **Live System Logs:** A global "Logs" page streams structured backend logs (`log/slog`) in a terminal-style view over WebSocket, with level/search filtering, play/pause, and configurable SQLite-backed retention.

## 🔌 Multi-Provider Support
- **GitHub:** Full metadata extraction (Issues, Releases, Stars).
- **GitLab:** Core support for repository mirroring.
- **Generic Git:** Supports any valid Git URL for mirroring.
