# Convoke

Calendar and scheduling system for volunteer-run associations. Provides binding attendance commitments for shifts and meetings so organizations can plan with confidence.

## Prerequisites

- Go 1.25+
- Node.js (for Tailwind CSS)
- PostgreSQL 17+
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)
- Docker & Docker Compose (optional, for containerized dev)

## Quick Start (Docker)

```sh
docker compose up
```

App at `http://localhost:8080`. PostgreSQL on port 5432. Dev mode enabled by default.

## Local Development

```sh
# 1. Start PostgreSQL (or use Docker for just the DB)
docker compose up -d postgres

# 2. Copy env and adjust if needed
cp .env.example .env

# 3. Install Node dependencies (Tailwind + DaisyUI)
npm install

# 4. Run in dev mode (CSS watch + Go server with hot reload)
make dev
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make dev` | CSS watch + Go server in dev mode |
| `make build` | Compile CSS + build binary to `bin/convoke` |
| `make run` | Run server without dev mode |
| `make css` | Build CSS once |
| `make sqlc` | Regenerate Go code from SQL queries |
| `make test` | Run unit tests |
| `make e2e` | Run Playwright e2e tests (requires Docker) |
| `make clean` | Remove build artifacts |

## Environment Variables

See `.env.example`. Key variables:

- `DATABASE_URL` — PostgreSQL connection string
- `LISTEN_ADDR` — Server bind address (default `:8080`)
- `DEV_MODE` — Enable dev mode (fake auth bypass, template reloading)
- `ADMIN_GROUP` — IdP group name for association admins
- `CSRF_SECRET` — HMAC secret for CSRF tokens
