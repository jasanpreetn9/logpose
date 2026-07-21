# Logpose frontend data reference

Every data field currently available from the backend, grouped by the page/view it belongs to. Field names are the frontend (camelCase) names as bound in `frontend/src/lib/types.ts` unless noted. Use this as the source of truth for what a redesigned UI has to work with — it doesn't prescribe layout, just the raw and computed values available.

See [Actions and mutations](#actions-and-mutations) at the bottom for every write/trigger the UI can perform, with its required inputs.

---

## Global / always-on-screen

| Field | Type | Source | Notes |
|---|---|---|---|
| `appVersion` | string | `GET /api/version` | e.g. `"v0.3"`. Shown in sidebar today. |
| Health checks | array — see [Health banner](#health-banner) | `GET /api/health`, polled every 60s | Only rendered when non-empty. |
| Arc list (for nav) | `arc`, `title`, `episodeCount`, `episodesDownloaded` | derived from the Library payload | Used to build a sidebar arc list with per-arc progress. |
| SSE live events | same shape as [Activity event](#activity--history) | `GET /api/events` (SSE stream) | Pushes new activity/import events into the UI without polling. |

---

## Library page (arc grid)

One card per arc.

| Field | Type | Example | Notes |
|---|---|---|---|
| `arc` | number | `36` | Arc number, 1-indexed. |
| `title` | string | `"Egghead"` | |
| `status` | string | `"WIP"` / `"TBR"` / `""` | Work-in-progress / to-be-released flag from the metadata source. |
| `resolution` | string | `"1080p"` | |
| `audioLanguages` | string | `"JA, EN, DE"` | Comma-separated. |
| `subtitleLanguages` | string | `"EN, AR, DE, FI..."` | Comma-separated, can be long (10+ languages) — needs truncation/expand affordance. |
| `mangaChapters` | string \| null | `"1005-1006"` | |
| `numberOfChapters` | string \| null | `"7"` | |
| `animeEpisodes` | string \| null | `"1023-1025"` | Original anime episode numbers the arc adapts. |
| `episodesAdapted` | string \| null | `"5"` | |
| `fillerEpisodes` | string \| null | `""` | Often empty. |
| `timeSavedMins` | string \| null | `"28"` | Minutes saved vs. watching the original anime run. |
| `timeSavedPercent` | string \| null | `"27.00"` | Already stripped of the `%` sign — append it in the UI. |
| `episodeCount` | number | `21` | Total episodes in this arc per metadata. |
| `episodesDownloaded` | number | `9` | Drives a progress bar / "9 / 21" badge. |
| **Not yet wired up:** arc poster image | — | metadata source has one, Logpose doesn't expose it yet | Worth requesting as a backend addition if the redesign wants poster art — arc folders on disk already have `season##-poster.jpg` files. |

**Derived for display:**
- Progress fraction: `episodesDownloaded / episodeCount`
- Status label: "Not Started" / "In Progress" / "Complete" (bucketed from the progress fraction — not a raw field)

---

## Arc detail page (episode list within an arc)

Arc header repeats the fields above. Then a list of episodes:

| Field | Type | Example | Notes |
|---|---|---|---|
| `arc` | number | `36` | |
| `episode` | number | `21` | |
| `title` | string | `"Luffy vs. Kizaru"` | |
| `description` | string | `"Luffy faces off in a spectacular aerial battle..."` | Can be 1–3 sentences. |
| `released` | string | `"2026-06-15"` | ISO date. |
| `downloaded` | boolean | `true` | True if *any* version is imported or queued (see versions below). |
| `monitored` | boolean | `true` | Episode-level flag, independent of version. |
| `versions` | array of `EpisodeVersion` | — | Usually 1 entry (`normal`), sometimes 2 (`normal` + `extended`). |

### `EpisodeVersion` (nested under each episode)

| Field | Type | Example | Notes |
|---|---|---|---|
| `crc32` | string | `"DBDB2978"` | 8-char hex checksum, uniquely identifies the release file. |
| `version` | `"normal"` \| `"extended"` | `"normal"` | |
| `filePath` | string \| null | `"/media/library/36 - Egghead/S36E21..."` | Only set once imported. |
| `status` | `"imported"` \| `"missing"` \| `"upgradable"` \| `"queued"` | `"imported"` | See status meanings below. |

**Status meanings** (for badge/color design):
- `missing` — not downloaded, no version present.
- `queued` — a download for this exact version is in progress in qBittorrent right now.
- `imported` — this version is the one currently in the library.
- `upgradable` — a *different* version of this episode is imported, and this one (usually `extended`) is available as an upgrade.

**Actions available per episode:** toggle monitored, download (single version), view details (opens a modal with the full description + file path).

---

## Wanted page

Filtered/derived view over the Library data — no separate endpoint. Shows only episodes where `monitored && (missing or upgradable or queued)`.

| Field (per row) | Source |
|---|---|
| `arc`, `episode`, `title` | from the episode |
| Arc `title` | from the parent arc |
| Status badge: "Missing" / "Upgrade Available" / "Queued" | derived from `versions[].status` |
| Action button: "Download" / "Upgrade" / disabled "Queued" | derived the same way |

---

## Queue page

One row per torrent currently in the `logpose` qBittorrent category.

| Field | Type | Example | Notes |
|---|---|---|---|
| `hash` | string | `"a1b2c3..."` | qBittorrent torrent hash, used for the remove action. |
| `name` | string | `"[One Pace][1093-1094] Egghead 21 [1080p][DBDB2978].mkv"` | Raw torrent/file name — fallback display when it can't be matched to an episode. |
| `title` | string | `"Luffy vs. Kizaru"` | Empty string if the name couldn't be resolved to a known episode. |
| `arc` | number | `36` | `0` if unresolved. |
| `episode` | number | `21` | `0` if unresolved. |
| `progress` | number | `0.42` | Fraction 0–1. Multiply by 100 for a percent. |
| `size` | number | `1552428800` | Bytes — needs formatting (KiB/MiB/GiB). |
| `dlspeed` | number | `2097152` | Bytes/sec — format as `.../s`; `0` means stalled/paused. |
| `eta` | number | `340` | Seconds. Values `< 0` or `>= 8640000` mean "unknown" (qBittorrent's sentinel for infinite/no ETA) — render as `—`, not a huge duration. |
| `state` | string | `"downloading"` | Raw qBittorrent state string — see below. |

**Known `state` values worth designing distinct treatments for:** `downloading`, `stalledDL`, `metaDL`, `queuedDL`, `pausedDL`, `forcedDL`, `checkingDL`, `uploading`, `stalledUP`, `checkingUP`, `error`, `missingFiles`, `allocating`. In practice you'll mostly see `downloading`, `stalledDL`, and occasionally `error`.

**Action available per row:** remove from qBittorrent (does not delete downloaded files).

### Unmatched downloads (same page)

Files sitting in the downloads folder that couldn't be auto-matched to an episode.

| Field | Type | Example | Notes |
|---|---|---|---|
| `path` | string | `"/downloads/random_file.mkv"` | Full path, used when submitting a manual match. |
| `name` | string | `"random_file.mkv"` | Display name. |
| `reason` | `"unparseable"` \| `"unknown_crc"` | `"unknown_crc"` | Why it wasn't auto-imported. |
| `crc32` | string (optional) | `"DEADBEEF"` | Only present when `reason` is `unknown_crc`. |

**Flow:** user picks an arc → episode → version (normal/extended) from dropdowns populated off the Library data, which calls a preview endpoint returning:

| Field | Type | Example |
|---|---|---|
| `title` | string | `"Romance Dawn, the Dawn of an Adventure"` |
| `arc` | number | `1` |
| `episode` | number | `1` |
| `version` | string | `"normal"` |
| `destFolder` | string | `"Romance Dawn"` |
| `destFilename` | string | `"S01E01 - Romance Dawn, the Dawn of an Adventure [A1B2C3D4].mkv"` |

This is shown in a confirm modal (old name struck through → new name) before the user commits the import — same pattern as Rename Files below.

---

## Rename Files (modal, triggered from the header)

Preview-then-confirm, not a page. Returned as a flat list plus a total:

| Field | Type | Example |
|---|---|---|
| `folder` | string | `"36 - Egghead"` |
| `from` | string | `"[One Pace][1093-1094] Egghead 21 [1080p][DBDB2978].mkv"` |
| `to` | string | `"S36E21 - Luffy vs. Kizaru [DBDB2978].mkv"` |
| `total` (summary, not per-row) | number | `34` — total recognized files scanned, vs. the count actually needing a rename |

---

## Activity page / History page

Same event shape, two different filtered views (History = only `import` events).

| Field | Type | Example | Notes |
|---|---|---|---|
| `id` | string | `"482"` | Monotonic, string-typed. |
| `type` | enum | `"import"` | One of: `download_queued`, `download_failed`, `library_scan`, `downloads_scan`, `import`, `metadata_refresh`. |
| `timestamp` | string | `"2026-07-21T18:24:50Z"` | ISO 8601 UTC. |
| `message` | string | `"Imported: Luffy vs. Kizaru"` | Human-readable headline. |
| `details` | string | `"/media/library/36 - Egghead/S36E21..."` | Secondary line — usually a file path, a URL, or an error message depending on `type`. |
| `success` | boolean | `true` | Drives an icon/color (checkmark vs. error). |

**Design note:** `details` is heterogeneous by `type` — for `import` it's a file path, for `download_queued` it's a torrent URL (possibly a very long magnet URI now — see below), for a failure it's the error text. Consider a monospace/truncated treatment for `details` rather than one universal style.

---

## Settings page

Grouped sections; all fields round-trip through `GET/POST /api/config`.

### Paths
| Field | Type |
|---|---|
| `libraryPath` | string |
| `downloadPath` | string |
| `libraryJsonPath` | string |

### Metadata
| Field | Type | Notes |
|---|---|---|
| `metadataEpisodesUrl` | string | |
| `metadataArcsUrl` | string | |
| `metadataRefreshInterval` | string | Go duration format, e.g. `"24h"`, `"30m"`. |
| Refresh result (action feedback, not stored state) | `episodes`, `arcs`, `nfosUpdated`, `grabbed`, `lastUpdated` (numbers/string) | Shown after clicking "Refresh Now" — e.g. "Refreshed — 529 episodes, 36 arcs, 2 NFOs regenerated, 3 auto-grabbed." |

### qBittorrent
| Field | Type | Notes |
|---|---|---|
| `qbEnabled` | boolean | |
| `qbHost` | string | e.g. `"http://127.0.0.1:8080/"` |
| `qbUsername` | string | |
| `qbPassword` | **write-only** | Never returned by GET; UI keeps a separate blank field with "leave blank to keep current" placeholder. |
| Test Connection result | `ok` (boolean), `version` (string, optional), `error` (string, optional) | Action feedback, not stored. |

### Automation
| Field | Type | Notes |
|---|---|---|
| `autoDownload` | boolean | Auto-queues monitored missing episodes after every metadata refresh. |

### Notifications
| Field | Type | Notes |
|---|---|---|
| `discordWebhookUrl` | string | Blank disables Discord notifications entirely. |
| `jellyfinUrl` | string | Blank disables the Jellyfin library-refresh call. |
| `jellyfinApiKey` | **write-only** | Same pattern as `qbPassword`. |

### Server
| Field | Type |
|---|---|
| `port` | string |

### Field-level validation
`POST /api/config` can return `{ errors: { fieldName: "message" } }` — every field above can have an inline error string keyed by its own name (e.g. `errors.metadataRefreshInterval = "must be a valid Go duration (e.g. 6h, 30m)"`).

---

## Health banner

Polled independently of Settings; renders as a dismissible strip, not a page.

| Field | Type | Example |
|---|---|---|
| `id` | string | `"qbittorrent"` \| `"metadata-stale"` \| `"auto-download-no-client"` \| `"library-path"` \| `"download-path"` |
| `level` | `"warning"` \| `"error"` | |
| `message` | string | `"qBittorrent unreachable at http://127.0.0.1:8080/: dial tcp: connection refused"` |

Zero or more checks can be active simultaneously — design should support stacking multiple banners, not just one.

---

## Data that exists in the metadata but isn't surfaced in any UI yet

Worth knowing about even though no current page shows them — useful if the redesign wants to add value without backend work:

| Field | Where it lives | Potential use |
|---|---|---|
| `magnet_uri`, `torrent_url` per episode file | backend `EpisodeFile.MagnetURI` / `.TorrentURL` | Currently used internally to pick the best download link; could show a small magnet/torrent icon on an episode row to indicate a direct (non-Nyaa-search) link is available. Only ~50% of the archive has these populated so far. |
| Arc poster image path | metadata source's `Arc.Poster`, not yet in Logpose's own model | Arc grid could use real poster art instead of text-only cards — library folders already have `season##-poster.jpg`/`.png` files on disk. |
| Manga chapter / anime episode cross-references | already in Arc and Episode data (`mangaChapters`, `animeEpisodes` etc.) | Could power a "compare to source" detail view. |

---

## Actions and mutations

Every write or trigger the UI can perform, grouped by page. "Inputs" are what the user has to supply or select; "Result" is what the UI should reflect back after it succeeds. None of these return the updated entity directly — after most of them, the frontend re-fetches the episode/arc list to pick up the change (noted per action).

### Episode-level (Arc detail page)

| Action | Inputs | Result / feedback | Notes |
|---|---|---|---|
| Toggle monitored | `arc`, `episode`, `monitored` (bool) | Episode's `monitored` flips immediately | No confirmation needed — this is a low-stakes toggle. |
| Download / Upgrade | `crc32` of the target version | Queues in qBittorrent; version's `status` should optimistically move toward `queued` | Fails with "no download URL available" if metadata has no URL for that release — surface this distinctly from a network error, it means the release genuinely isn't downloadable yet. |
| View details | none (opens modal for an already-loaded episode) | Read-only modal — description, released date, file path if imported | |

### Arc-level (Arc detail page header)

| Action | Inputs | Result / feedback | Notes |
|---|---|---|---|
| Monitor Arc / Unmonitor Arc | `arcId`, `monitored` (bool) | Sets every episode in the arc to that monitored state | This is a bulk action — worth a distinct visual weight from the per-episode toggle since it can affect dozens of episodes at once. |
| Download Monitored | `arcId` (path only, no body) | `{ queued, total }` — e.g. "Queued 4 of 7 monitored episodes" | Skips anything already imported or without a URL; the count discrepancy (`queued < total`) is expected and shouldn't read as a failure. |
| Verify NFOs | `arcId` (path only) | `{ updated, total }` — e.g. "12 of 12 NFOs verified" | Regenerates Jellyfin sidecar files for everything already imported in the arc; low-frequency maintenance action. |

### Header-level (global, available from every page)

| Action | Inputs | Result / feedback | Notes |
|---|---|---|---|
| Scan Library | none | `{ filesFound, filesMarkedMissing }` | Walks the on-disk library and reconciles it with the database — for when files were moved/deleted outside the app. |
| Scan Downloads | none | `{ imported }` — count of files moved into the library | Normally automatic (a folder watcher triggers this), so this is a manual "check now" fallback. |
| Refresh Metadata | none | `{ episodes, arcs, nfosUpdated, grabbed, lastUpdated }` | `grabbed > 0` means autoDownload queued new episodes as a side effect of this click — worth calling out distinctly, e.g. "3 auto-grabbed," since it's a consequence the user didn't explicitly ask for. |
| Rename Files | *(two-step)* | Step 1 (preview, no inputs): list of `{ folder, from, to }` + `total`. Step 2 (confirm, no additional inputs — replays the same plan): `{ renamed, total }` | Never let step 2 fire without the user seeing step 1's list first — this touches every mis-named file on disk in one action. |

### Queue page

| Action | Inputs | Result / feedback | Notes |
|---|---|---|---|
| Remove from queue | `hash` | Row disappears from the queue list | Does **not** delete the downloaded files — the copy should say so explicitly (e.g. "Remove torrent (keeps files)"), since "remove" is easy to misread as destructive. |
| Match (unmatched file) → Preview | `path`, `arc`, `episode`, `version` (picked from dropdowns) | Opens the confirm modal with resolved `title`, `destFolder`, `destFilename` | Pure read — no files touched yet. |
| Confirm import | same `path`/`arc`/`episode`/`version` as the preview | File moves into the library; row disappears from Unmatched; episode's `status` becomes `imported` | The only step that actually writes to disk — must be gated behind the preview modal, never a single click from the unmatched list. |

### Settings page

| Action | Inputs | Result / feedback | Notes |
|---|---|---|---|
| Save Settings | full config form (see [Settings page](#settings-page) fields) | `{ status: "ok" }` or `{ errors: {...} }` per-field | `qbPassword`/`jellyfinApiKey` are omitted from the request entirely when left blank (not sent as empty strings) — sending an empty string would be indistinguishable from "clear the secret," which isn't the intent. |
| Test Connection (qBittorrent) | `host`, `username`, `password` (from the in-progress form, not necessarily saved yet) | `{ ok, version }` or `{ ok: false, error }` | Lets the user validate credentials *before* saving — should not require Save first. |
| Refresh Now (metadata) | none | same shape as header-level Refresh Metadata | Duplicate entry point for the same action — keep the feedback copy consistent between the two locations. |

### Implicit / background (not user-triggered, but affect what's on screen)

| Trigger | Effect |
|---|---|
| Metadata refresh ticker (interval from `metadataRefreshInterval`) | Same as manual Refresh Metadata — including auto-grab if enabled. UI should reflect new "grabbed" episodes appearing in Wanted/Queue without a page reload (SSE or poll-driven). |
| qBittorrent completion poller (~10s) | Torrents that finish move from Queue into the library automatically — a `queued` episode version transitions straight to `imported`. |
| Downloads-folder watcher | New files dropped into the download path trigger an automatic downloads scan — same effect as clicking Scan Downloads. |
| Health poll (60s) | Banner appears/disappears without user action. |
