# InhousePredictor

InhousePredictor is a private, play-money prediction market for teams.

Employees create binary markets, spend internal points on `YES` or `NO`, watch prices move as new information comes in, and review outcomes once a resolver records evidence. The product is designed to feel like a trustworthy internal forecasting tool, not a trading terminal.

## What It Does

- lets employees create markets about company questions, delivery confidence, SLAs, hiring, and similar internal forecasts
- uses internal points instead of real money
- updates market prices in real time
- shows portfolios, open positions, and resolved outcomes
- requires visible resolver identity and evidence on resolution
- supports disputes and an admin dispute-review queue

## Main Features

- email/password auth with JWT + auth cookie support
- market browsing with filters and live price updates
- market detail with trade ticket, evidence, and dispute state
- YES / NO trading with buy and sell support powered by a constant-product market maker
- portfolio view with forecast accuracy and points earned / lost
- create market flow with resolver and timing rules
- resolver flow for recording outcomes with evidence
- admin dispute review flow

## Tech Stack

- Backend: Go, chi, pgx, PostgreSQL, gorilla/websocket
- Frontend: React, TypeScript, Vite, React Router
- Infra: Docker, Docker Compose
- Testing: Go `testing` + `testify`

## Repository Layout

```text
cmd/server/          Go API entrypoint
internal/            Backend packages
frontend/            React frontend
migrations/          PostgreSQL schema
docker-compose.yml   Local Postgres + API stack
TESTING.md           Test philosophy and commands
ARCHITECTURE.md      Deeper technical overview
```

## Prerequisites

You need:

- Go 1.24+
- Node.js 20+
- npm 10+
- Docker + Docker Compose

## Quick Start

### 1. Clone and configure

```bash
git clone <your-repo-url>
cd inhousePredictor
cp .env.example .env
```

Set at least:

- `JWT_SECRET` to a long random value
- optionally `BOOTSTRAP_ADMIN_EMAIL` if you want the first matching user to be promoted to admin on startup

### 2. Start Postgres and the API

```bash
docker compose up --build -d
```

This starts:

- API at `http://localhost:8080`
- PostgreSQL at `localhost:5432`

### 3. Start the frontend

In a second terminal:

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

Open:

- Frontend: `http://localhost:5173`

The Vite dev server proxies API requests to `http://localhost:8080`.

## Production Build

### Backend

```bash
go build ./...
```

### Frontend

```bash
cd frontend
npm run build
```

The frontend production bundle is written to `frontend/dist/`.

## Environment Variables

From `.env.example`:

- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - signing key for auth tokens
- `PORT` - API port
- `ALLOWED_ORIGINS` - comma-separated browser origins allowed for WebSocket connections
- `BOOTSTRAP_ADMIN_EMAIL` - optional email to auto-promote as the first admin when no admin exists yet

## First Admin Setup

Two options:

1. Set `BOOTSTRAP_ADMIN_EMAIL` before starting the API
2. Promote a user directly in Postgres if needed

Example direct promotion:

```bash
docker compose exec postgres psql -U app -d inhousepredictor -c \
  "UPDATE users SET is_admin = true WHERE email = 'you@example.com';"
```

Log out and back in after promotion so the new admin claim is reflected in the token/cookie.

## Running Tests

Backend:

```bash
go test ./...
```

See `TESTING.md` for more detail.

## Useful Commands

Start stack:

```bash
docker compose up --build -d
```

Stop stack:

```bash
docker compose down
```

API logs:

```bash
docker compose logs -f api
```

Database logs:

```bash
docker compose logs -f postgres
```

Frontend dev server:

```bash
cd frontend
npm run dev -- --host 0.0.0.0
```

## Product Rules Worth Knowing

- creators cannot trade their own markets
- resolvers must attach evidence when resolving a market
- disputes are time-bounded
- the platform is play-money only
- users can buy and sell positions before market resolution; sell orders currently use whole-share amounts

## Additional Docs

- `ARCHITECTURE.md` - backend structure and system mechanics
- `API.md` - request/response contract
- `CHANGELOG.md` - release-facing change history
- `CONTRIBUTING.md` - local setup and contribution workflow
- `VERSION` - current project version label
- `DATA_MODEL.md` - domain entities, invariants, and schema notes
- `DESIGN.md` - product design system and screen blueprints
- `USER_FLOWS.md` - core user journeys and UX expectations
- `TESTING.md` - backend testing philosophy and conventions
- `DECISIONS.md` - active product and technical decisions
- `COPY_GUIDE.md` - product voice and microcopy rules
- `PRD.md` - product goals and scope
- `PLAN.md` - phased implementation plan
- `QA.md` - release and verification guidance
- `SECURITY.md` - auth, trust boundaries, and abuse cases
- `OPERATIONS.md` - runtime and deployment guidance
- `TODOS.md` - prioritized implementation checklist
- `TODOS_SEC.md` - security-focused follow-up checklist
- `deep-research-report.md` - background research and market context
- `docs/` - supplemental roadmap and design notes

## Current Status

The backend, frontend foundation, core user flows, and admin dispute flow are implemented.

Remaining work is primarily around polish, QA hardening, and release readiness rather than basic product existence.
