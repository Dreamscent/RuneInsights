# RuneInsights

A self-hosted, single-user web dashboard for **RuneScape 3** hiscores. Add any player by
display name and watch their stats over time — levels, experience, XP to next level, XP gains
per day/week/month/year, benchmarked milestones, and per-skill training-rate estimates.

The official Jagex hiscores are polled on a schedule and stored locally in SQLite, so every
graph gets sharper the longer the server runs.

## Features

- **Player tracking** — add any RS3 player (standard, Ironman or Hardcore Ironman hiscore
  tables). Snapshots are captured automatically on a per-player schedule (15 min – 4 h) and
  stored locally; manual refresh is capped at once every 5 minutes.
- **Skills dashboard** — the core view: current level (plus virtual level), XP, global rank,
  progress bar to the next level, XP/day pace, sortable and filterable.
- **Milestones** — automatically computed next milestone per skill: level 99, 110, 120 and the
  200M XP cap. Invention (elite skill, own XP curve) additionally tracks level 150, exactly.
- **XP calculator** — every skill drawer shows time to next level and time to the next
  milestone for any recorded training method, with live recalculation.
- **Training rates** — a human-editable CSV (`data/skill_rates.csv`) maps skills to training
  methods and expected XP per hour. Skills can hold multiple methods; the first is the default.
  The dashboard ships a GUI page that edits this same file, with add / remove / set-default
  controls and hot reload.
- **Progress** — XP gained per day / week / month / year, per-skill breakdown, and XP-over-time
  charts for any skill or timeframe (24 h to 1 y). Periods without enough history fall back to
  total XP since tracking began, annotated with the tracked-day count.
- **Bosses & minigames** — scores, ranks and period deltas.
- **Leaderboard** — top 50 on any skill, straight from the official ranking endpoint.
- **Clan lookup** — browse any clan's member list and track members with one click.
- Polished dark UI (Svelte 5, Tailwind CSS 4, Chart.js), responsive down to mobile.

## How it works

```
Jagex official APIs ──▶ Go server ──▶ SQLite (snapshots) ──▶ REST/JSON ──▶ Svelte SPA
      (rate-limited        ▲                                               │
        ~1 req/s)          └── background scheduler (per-player intervals) ◀┘
```

All upstream calls are made server-side because Jagex's APIs don't send CORS headers. Use of
the endpoints:

| What | Endpoint |
| --- | --- |
| Player hiscores | `secure.runescape.com/m=hiscore/index_lite.json?player=X` (+ ironman / hardcore variants) |
| Top-50 rankings | `secure.runescape.com/m=hiscore/ranking.json` |
| Clan members | `secure.runescape.com/m=clan-hiscores/members_lite.ws?clanName=X` |

Only the latest snapshot per UTC day is retained, bounding database growth.

## Installation

Requirements:

- [Go](https://go.dev/dl/) 1.26 or newer
- [Node.js](https://nodejs.org/) 20 or newer (a one-time build of the web frontend)

```sh
git clone <your-repository-url> runeinsights
cd runeinsights

# 1. build the frontend (once; only needed again when UI changes)
(cd web && npm install && npm run build)

# 2. build the server (single static binary)
go build -o runeinsights ./cmd/server
```

### Serving it

```sh
./runeinsights
```

- The dashboard is served on **all network interfaces** (host `0.0.0.0`) at
  **http://<your-host>:8080** — reachable from other machines/devices, not only localhost.
  Bind it to `127.0.0.1` instead if you prefer localhost-only.
- Data and settings live in `./data/` (SQLite database) and `config.json`.
- Stop with `Ctrl-C`; shutdown is graceful.

Run it behind systemd, Docker or any process supervisor for a permanent setup — see below.

For frontend development with hot reload:

```sh
(cd web && npm run dev)   # proxies /api requests to localhost:8080
```

### Configuration

All before-run settings live in `config.json` (created next to the server, tracked in the
repository) and can be **overridden per-invocation with environment variables**:

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "dbPath": "./data/rs.db",
  "staticDir": "./web/dist",
  "skillRatesPath": "./data/skill_rates.csv"
}
```

| Setting | env var | Default | Meaning |
| --- | --- | --- | --- |
| `host` | `HOST` | `0.0.0.0` | Interface to bind (`127.0.0.1` = localhost only) |
| `port` | `PORT` | `8080` | HTTP listen port |
| `dbPath` | `DB_PATH` | `./data/rs.db` | SQLite database file (created automatically) |
| `staticDir` | `STATIC_DIR` | `./web/dist` | Built frontend; falls back to an API-only notice if missing |
| `skillRatesPath` | `SKILL_RATES_PATH` | `./data/skill_rates.csv` | Training-rate file (hot-reloaded) |

A different config file path can be supplied via `CONFIG` (e.g.
`CONFIG=/etc/runeinsights/config.json ./runeinsights`). Missing file, missing keys and even
malformed JSON all fall back to sane defaults — the server always starts.

### systemd (recommended for a VPS)

```ini
[Unit]
Description=RuneInsights
After=network-online.target

[Service]
User=youruser
WorkingDirectory=/opt/runeinsights
ExecStart=/opt/runeinsights/runeinsights
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

`sudo systemctl enable --now runeinsights`, then open `http://<host>:8080`.

### Docker

```sh
docker build -t runeinsights .
docker run -d -p 8080:8080 -v rs-data:/home/app/data runeinsights
```

## Training rates

Edit `data/skill_rates.csv` (or use the **Training rates** page) to record how fast you train
each skill:

```
skill,exp_per_hour,training_method
Hunter,350000,Afk Croesus Front
Hunter,1200000,Chinchompas (justicar armour)
```

Multiple entries per skill are supported; the **first** entry is the default used everywhere.
Changes (file or GUI) are picked up automatically with no restart. Skills at 0 — or missing —
show "no estimate".

## REST API

```
GET    /api/players                          list tracked players
POST   /api/players                          {"name","accountType"} — validates live, stores first snapshot
PATCH  /api/players/{id}                     {"accountType"?,"intervalMin"?}
DELETE /api/players/{id}
POST   /api/players/{id}/refresh             manual refresh (≥5 min gap)
GET    /api/players/{id}/skills              levels, xp, ranks, xp-to-next, rates, milestones
GET    /api/players/{id}/rates?period=       day|week|month|year gains per skill + overall
GET    /api/players/{id}/history?skill=&from=&to=   XP time series
GET    /api/players/{id}/activities[?period=]       boss/minigame scores (+gains)
GET    /api/skill-rates                      grouped training-rate data
PUT    /api/skill-rates/{skill}              {"methods":[{perHour,method}, ...]}
GET    /api/leaderboard?table=&category=     top-50 ranking proxy
GET    /api/clans/{name}                     clan member list
GET    /api/healthz                          liveness
```

## Project layout

```
cmd/server/        entrypoint (config loading, HTTP server, static hosting, graceful shutdown)
internal/config/   config.json loader (defaults ← file ← environment)
internal/xp/       experience <-> level math (standard + elite/Invention curves, wiki-verified)
internal/hiscore/  Jagex API clients (hiscores, rankings, clan members) + rate limiting
internal/db/       SQLite schema + queries (players, snapshots, skills, activities)
internal/tracker/  polling scheduler: per-player intervals, snapshot writing, daily pruning
internal/stats/    gain/rate computation from snapshot history
internal/rates/    training-rate CSV store (hot reload, multiple methods per skill)
internal/api/      REST handlers (JSON)
web/               Svelte 5 + Vite + Tailwind CSS dashboard
```

## Notes and caveats

- Invention is an elite skill with its own XP curve (99 = 36,073,511 XP, virtual cap 150);
  its levels, progress and milestones are computed from the official elite table.
- Combat level is computed locally with the current RS3 formula (max 152).
- Upstream rate limits are undocumented; the client keeps a conservative ~1 request/second
  global limit with backoff.
- Only XP/values from the hiscores are captured; nothing is fabricated for missing data.

## Verification

```sh
go build ./... && go vet ./...   # backend
(cd web && npm run build)        # frontend
(cd web && npx svelte-check)     # optional full type check
```

XP math is unit-tested against the anchors published on the RuneScape wiki
(level 99 = 13,034,431 XP, level 120 = 104,273,167 XP, virtual 126 = 188,884,740 XP;
Invention 99 = 36,073,511 XP from the official elite table).
