---
sidebar_position: 3
title: Setup & Installation
---

# Setup & Installation ⚙️

GitPatrol is designed for simple, robust deployment within home labs or corporate infrastructure.

## Prerequisites
- **Docker & Docker Compose** (Recommended)
- **Git CLI** (1.7+ or higher)
- **Go 1.24+** (for manual backend builds)
- **Bun** (for manual frontend development)

---

## 🐋 Docker Installation (Recommended)
Running GitPatrol via Docker Compose is the easiest way to ensure all dependencies are correctly configured.

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/EmielDehaen/gitpatrol.git
    cd gitpatrol
    ```
2.  **Environment Configuration:**
    The backend automatically generates secrets and sensible defaults on first run and writes them to `gitpatrol.env` next to the database, but you can override any of these via your environment or that file:

    | Variable | Description | Default |
    |----------|-------------|---------|
    | `JWT_SECRET` | Secret key for signing session JWTs. | Randomly generated |
    | `PASSWORD_PEPPER` | Pepper mixed into password hashes. | Randomly generated |
    | `DB_PATH` | Path to the SQLite database file. | `./db/gitpatrol.db` |
    | `DATA_DIR` | Root directory for Git mirrors, avatars, and archived metadata. | `./data` |
    | `PORT` | Port the backend HTTP server listens on. | `8080` |
    | `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed frontend origins. | `http://localhost:5173,http://localhost:3000` |
    | `WORKERS` | Number of concurrent sync workers. | `3` |
    | `LOG_RETENTION_DAYS` | Days to keep rows in the system logs table. | `7` |
    | `LOG_MAX_ROWS` | Maximum system log rows kept, whichever limit hits first. | `10000` |
    | `GITHUB_TOKEN` | GitHub token used for private repos and to avoid API rate limits. | none |
    | `GITLAB_URL` | Base URL of your GitLab instance. | `https://gitlab.com` |
    | `GITLAB_TOKEN` | GitLab token used for private repos and to avoid API rate limits. | none |
    | `GITEA_URL` / `GITEA_TOKEN` | Base URL and token for a Gitea recovery vault. | none |
    | `EXPORT_DESTINATION` | Default one-click export target (`github`, `gitlab`, or `gitea`). | none |
    | `GP_SECURE_COOKIE` | Set to `true` to enable the Secure flag on the session cookie. | *(planned — not implemented yet)* |

3.  **Spin up the containers:**
    ```bash
    docker-compose up -d
    ```
4.  **Bootstrap the System:**
    Open `http://localhost:3000`. If it's your first time, you will be prompted to create the primary administrator account.

---

## 🛠️ Manual Development Setup

### Backend (Go)
1.  **Enter the directory:** `cd backend`
2.  **Run the application:** `go run .`
    The backend runs on port `:8080` by default and stores data in `./db` and `./data`.

### Frontend (SvelteKit)
1.  **Enter the directory:** `cd frontend`
2.  **Install dependencies:** `bun install`
3.  **Start the development server:** `bun run dev`
    The dashboard will be available at `http://localhost:5173`. Ensure it points to the correct `PUBLIC_API_URL`.

## 📦 Data Volumes
If running in Docker, ensure the following volumes are mapped to persistent storage.

**`docker-compose.yml` (development image):**
- `/app/db/`: Contains `gitpatrol.db` (SQLite) and `gitpatrol.env`.
- `/app/data/`: Contains physical Git mirrors, avatars, and archived metadata.

**`docker-compose.production.yml` (single-binary image):**
- `/var/lib/gitpatrol/`: A single mount containing both `db/` and `data/` (set via `DB_PATH` and `DATA_DIR` in the image).
