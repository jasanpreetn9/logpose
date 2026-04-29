# Logpose

A Sonarr-style media manager built specifically for [One Pace](https://onepace.net). Tracks arcs and episodes, monitors for new releases, automatically downloads through qBittorrent, generates Jellyfin-compatible NFO files, and keeps your library organised.

<div align="center">
  <img src="https://github.com/jasanpreetn9/onepace-library/blob/main/public/view.png?raw=true">
</div>

---

## Features

**Library management**
- Scans your media directory and imports recognised One Pace files
- Supports both filename formats:
  - Download: `[One Pace][1058-1059] Egghead 01 [1080p][CA3F14A8].mkv`
  - Library: `S36E01 - New Emperors [CA3F14A8].mkv`
- Generates Jellyfin-compatible `.nfo` sidecar files
- SQLite-backed persistent state (survives restarts)

**Monitoring & downloads**
- Monitor individual episodes or entire arcs with one click
- Wanted page lists all monitored-but-missing episodes and available upgrades
- One-click download queues torrents directly to qBittorrent
- Automatic import poller: completed torrents are moved and catalogued without manual scans
- fsnotify watcher auto-triggers a scan when files appear in the downloads folder
- Multi-file (arc pack) torrents are walked and each episode imported individually

**Upgrades**
- Detects when a newer version of an already-downloaded episode is available
- Surfaces upgradable episodes on the Wanted page with a dedicated Upgrade button

**Metadata**
- Fetches arc and episode metadata from [`jasanpreetn9/onepace-metadata`](https://github.com/jasanpreetn9/onepace-metadata)
- Configurable refresh interval; NFOs for changed episodes are regenerated automatically

**UI**
- Light / dark / system theme with persistent preference
- Arc grid with download progress bars and status badges
- Per-arc episode list with monitor toggles, version details, and inline download/upgrade buttons
- Activity feed and import history via SSE (live, no polling)
- Settings page to update all config values at runtime

---

## Docker (recommended)

### 1. Configure

Copy the sample config and edit it:

```bash
cp config.yml config.yml   # already at the repo root
```

Edit `config.yml`:

```yaml
port: "8989"

libraryPath: "/media/library"      # path INSIDE the container (see volumes below)
downloadPath: "/media/downloads"

libraryJsonPath: "./data/library.json"

metadata:
  episodesUrl: "https://raw.githubusercontent.com/jasanpreetn9/onepace-metadata/refs/heads/main/data/episodes.json"
  arcsUrl: "https://raw.githubusercontent.com/jasanpreetn9/onepace-metadata/refs/heads/main/data/arcs.json"

metadataRefreshInterval: "24h"

qbittorrent:
  enabled: true
  host: "http://your-qbittorrent-host:8080/"
  username: "admin"
  password: "adminadmin"
```

### 2. Edit volume paths in `docker-compose.yml`

Open `docker-compose.yml` and update the two media volume lines to match your actual paths:

```yaml
- /your/library/path:/media/library
- /your/downloads/path:/media/downloads
```

### 3. Start

```bash
docker compose up -d
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8989

### 4. Update

```bash
docker compose pull
docker compose up -d --build
```

---

## Manual setup

### Prerequisites

- Go 1.21+
- Node.js 20+

### Backend

```bash
cd backend
go run ./cmd/server
```

The server reads `../config.yml` by default. Override with:

```bash
CONFIG_PATH=/path/to/config.yml DATA_DIR=/path/to/data go run ./cmd/server
```

### Frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server proxies `/api` to `http://localhost:8989` automatically.

---

## Project structure

```
onepace-library/
├── config.yml                    # User config (edit this)
├── data/                         # Runtime data (created automatically)
│   ├── library.db                # SQLite database
│   └── library.json              # JSON mirror / backup
├── docker-compose.yml
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/                  # HTTP handlers
│   │   ├── activity/             # Activity event store
│   │   ├── config/               # Config load/save
│   │   ├── db/                   # SQLite layer
│   │   ├── library/              # Library store (RWMutex + SQLite)
│   │   ├── metadata/             # Metadata fetch & cache
│   │   ├── nfo/                  # Jellyfin NFO generator
│   │   ├── poller/               # qBittorrent completion poller
│   │   ├── qbittorrent/          # qBittorrent Web API client
│   │   ├── scanner/              # Library & downloads scanner
│   │   ├── sse/                  # Server-Sent Events hub
│   │   └── watcher/              # fsnotify downloads watcher
│   ├── Dockerfile
│   └── go.mod
└── frontend/
    ├── src/
    │   ├── lib/
    │   │   ├── api.ts            # API wrapper
    │   │   ├── stores.ts         # Svelte stores
    │   │   ├── theme.ts          # Theme switcher
    │   │   └── types.ts          # Shared types
    │   └── routes/
    │       ├── +layout.svelte    # Sidebar, header, SSE
    │       ├── library/          # Arc grid + per-arc episode list
    │       ├── wanted/           # Missing & upgradable episodes
    │       ├── activity/         # Live activity feed
    │       ├── history/          # Import history
    │       └── settings/         # Runtime config editor
    ├── nginx.conf
    ├── Dockerfile
    └── package.json
```

---

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/episodes/all` | All arcs with episode versions and statuses |
| `POST` | `/api/scan/library` | Scan library directory |
| `POST` | `/api/scan/downloads` | Scan downloads directory |
| `POST` | `/api/download/:crc32` | Queue torrent in qBittorrent |
| `POST` | `/api/arcs/:id/monitor` | Monitor/unmonitor entire arc |
| `POST` | `/api/arcs/:id/download` | Download all monitored episodes in arc |
| `POST` | `/api/arcs/:id/verify-nfos` | Regenerate NFO files for arc |
| `POST` | `/api/episodes/:crc32/monitor` | Monitor/unmonitor single episode |
| `GET` | `/api/activity` | Recent activity events |
| `GET` | `/api/history` | Import history |
| `GET` | `/api/events` | SSE stream of live events |
| `GET` | `/api/config` | Current config |
| `POST` | `/api/config` | Update config at runtime |

---

## License

GNU GPLv3
