# TAM Backend

Backend for Telegram advertising marketplace (TAM) — connects channel owners with advertisers via escrow-style deals. REST API for Mini App, Telegram Bot for dialogs and auto-posting, optional User Bot and TON indexer.

## Table of Contents

- [Quick Start (Local Setup)](#quick-start-local-setup)
- [Full Configuration](#full-configuration)
- [Local Development](#local-development)
- [API](#api)
- [Environment Variables](#environment-variables)
- [License](#license)

## Quick Start (Local Setup)

**Prerequisites:**

- Go 1.24+
- Docker (for PostgreSQL)
- [Task](https://taskfile.dev/) — `brew install go-task` or `go install github.com/go-task/task/v3/cmd/task@latest`

**Steps:**

1. Copy env file:
   ```bash
   cp .env.example .env
   ```

2. Fill in `.env` (minimum for local run):
   - `DATABASE_DSN` — e.g. `postgres://username:password@localhost:5432/admarket?sslmode=disable`
   - `BOT_TOKEN` — from [BotFather](https://t.me/BotFather)
   - `WEB_APP_URL` — frontend URL (e.g. `http://localhost:5173` for local dev)
   - `JWT_SECRET` — any random string
   - `ADMINS_ID` — your Telegram user ID (e.g. via @userinfobot)

3. Start PostgreSQL:
   ```bash
   docker run --name my-postgres -e POSTGRES_USER=username -e POSTGRES_PASSWORD=password -e POSTGRES_DB=admarket -p 5432:5432 -d postgres
   ```

4. Run backend:
   ```bash
   task run
   ```

**Result:**
- API: http://localhost:8080/api/v1
- Swagger: http://localhost:8080/api/v1/swagger/index.html

---

**Alternative: one command** (creates DB, applies migrations, runs):
```bash
task restart
```

## Full Configuration

### User Bot (channel statistics)

User Bot fetches channel statistics via MTProto. To enable:

1. Create app at [my.telegram.org](https://my.telegram.org) → get `APP_ID`, `APP_HASH`
2. Set `PHONE_NUMBER`, `ADMINS_ID` (receives auth codes)
3. Message the Telegram Bot at least once (so it can send you codes)
4. Set `STATS_BOT=true`
5. Add User Bot as channel admin

Order: set `ADMINS_ID`, message Telegram Bot, then `STATS_BOT=true`.

### TON Indexer (deposit tracking)

- `INDEXER=true`
- `TON_MAIN_NET` — default `https://ton.org/global.config.json`
- `TON_MASTER_MNEMONIC` — 24-word wallet mnemonic

## Local Development

| Command | Description |
|---------|-------------|
| `task run` | Run backend |
| `task gen` | Generate Ent, enums, TS types, Swagger |
| `task migrate-create -- name` | Create new migration |
| `task migrate-rerun` | Reset DB and reapply migrations |
| `task db-remove` | Stop and remove Postgres container |
| `task restart` | Full restart (DB + migrations + run) |

**Docker:** `docker build -t tam-backend .` — build image only.

## API

**Base URL:** `/api/v1`  
**Swagger:** http://localhost:8080/api/v1/swagger/index.html

**Auth:** `POST /auth` with `{"init_data": "<telegram_init_data>"}` → JWT in `Authorization` header. All other endpoints require `Authorization: Bearer <token>`.

| Group | Main endpoints |
|-------|----------------|
| Customers | `GET /customers/me`, `PATCH /customers/me`, `GET /customers/me/deals`, `POST /customers/withdraw` |
| Channels | `POST /channels/add-channel`, `GET /channels/list-channels`, `GET /channels/:id`, `POST /channels/provide-channel-data`, `DELETE /channels/:id` |
| Deals | `POST /deals/offer-deal`, `GET /deals/:id`, `POST /deals/send-deal-data`, `POST /deals/make-deal`, `POST /deals/accept-deal`, `POST /deals/deal-message`, etc. |
| Briefs | `POST /briefs/add-brief`, `GET /briefs/:id`, `PATCH /briefs/:id`, `GET /briefs/list` |
| Upload | `POST /upload` |

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable | Used for |
|----------|----------|
| `APP_MODE` | Application mode (e.g. `local`) |
| `DATABASE_DSN` | PostgreSQL connection string |
| `SERVER_PORT` | HTTP port (default: 8080) |
| `ALLOWED_ORIGINS` | CORS origins (comma-separated or `*`) |
| `BOT_TOKEN` | Telegram Bot API token |
| `WEB_APP_URL` | Mini App frontend URL |
| `ADMINS_ID` | Telegram user IDs for User Bot auth codes (comma-separated) |
| `INIT_DATA_EXPIRE` | TTL for init_data (e.g. `6h`) |
| `JWT_SECRET` | JWT signing secret |
| `JWT_EXPIRE` | JWT expiry (e.g. `6h`) |
| `INDEXER` | `true` to enable TON indexer |
| `TON_MAIN_NET` | TON config URL |
| `TON_MASTER_MNEMONIC` | 24-word wallet mnemonic |
| `STATS_BOT` | `true` to enable User Bot |
| `APP_ID`, `APP_HASH` | User Bot (from my.telegram.org) |
| `PHONE_NUMBER` | User Bot auth |

## License

This project is licensed under the Custom Non-Commercial License.  
See the [LICENSE](LICENSE) file for details.
