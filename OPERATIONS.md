# InhousePredictor Operations Guide

> Last updated: 2026-03-27

---

## Purpose

This document defines how InhousePredictor should be run, bootstrapped, observed, and recovered.

The project is still early, so this guide covers both the current local-development reality and the target operating model required before first users.

---

## Runtime Overview

- API server written in Go 1.24
- PostgreSQL 16 as primary datastore
- WebSocket support for live updates
- Docker and Docker Compose for local startup

---

## Local Development

### Current Flow

```bash
cp .env.example .env
# edit .env and set JWT_SECRET
docker compose up --build -d

# in a second terminal
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

Expected local endpoints:

- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Frontend: `http://localhost:5173`

### Notes

- schema is reconciled on application startup
- frontend runtime is available through the Vite dev server in `frontend/`

---

## Environment Variables

### Required Today

- `JWT_SECRET`

### Required Before Launch

- `JWT_SECRET`
- `BOOTSTRAP_ADMIN_EMAIL` for first-admin bootstrap when needed
- any later DB or origin-allowlist settings required by deployment

---

## Startup Behavior

### Current State

- application runs migrations on startup before serving traffic
- startup fails clearly if migrations cannot be applied
- startup can bootstrap the first admin when configured

### Target State

- application runs migrations on startup before serving traffic
- startup fails clearly if migrations cannot be applied
- startup behavior stays consistent outside the current Docker-based local flow

---

## Database And Migrations

### Current State

- schema lives in `migrations/001_init.sql`
- `internal/migrate` runs schema reconciliation during normal startup

### Ongoing Improvements

- add future migrations through the same startup runner as schema evolves
- document migration rollback expectations for production deploys
- verify non-container deploys behave the same as local Docker startup

---

## Admin Bootstrap

### Current State

- if `BOOTSTRAP_ADMIN_EMAIL` is set and no admin exists, startup promotes that user automatically

### Target State

- if `BOOTSTRAP_ADMIN_EMAIL` is set and no admin exists, promote that user on startup
- promotion behavior is documented for both local and production environments

### Operational Reason

- first-run setup should not require raw SQL access

---

## Logging And Observability

### Current State

- structured JSON logging uses `slog`
- request logging runs through the shared HTTP middleware

### Target State

- use `slog`
- include structured fields for trade execution:
  - `trade_id`
  - `market_id`
  - `user_id`
  - `cost`
  - `shares`
  - `price_before`
  - `price_after`
- include payout logs with market-level context
- include auth or handshake failures where useful

### Rules

- never log plaintext passwords
- never log secrets

---

## Graceful Shutdown

### Current State

- graceful shutdown handles `SIGINT` and `SIGTERM`
- the HTTP server drains requests before exit and the WebSocket hub is shut down explicitly

### Target State

- catch `SIGTERM`
- call `server.Shutdown(ctx)` with drain window
- close WS connections cleanly where possible

### Reason

- restarts should not hard-drop all in-flight requests and sockets without cleanup

---

## Monitoring Priorities

Before first users, monitor at least:

- server start and migration success
- auth failures
- trade execution failures
- payout attempts
- dispute actions once implemented
- WS connection failures and origin/auth rejections

---

## Backups And Recovery

### Current State

- no explicit backup or restore process is documented yet

### Minimum Requirement Before Broader Rollout

- define Postgres backup cadence
- define restore procedure
- verify restore can recover markets, balances, trades, and disputes

### Operational Note

- because trades and payouts affect user trust directly, recovery procedures must preserve ledger integrity

---

## Rollback Strategy

### Current State

- no formal rollback procedure exists

### Minimum Requirement Before Launch

- document how to roll back application deploys safely
- document migration rollback expectations or compensating actions
- ensure payout and market-state changes are considered when evaluating rollback safety

---

## Deployment Notes

- CI/CD does not exist yet
- backend deployment remains the primary runtime concern today
- frontend build output exists, but production frontend deployment is still undocumented

---

## Operational Risks

- backup and restore remain undocumented
- frontend deployment expectations are still undocumented
- lack of documented backup or rollback increases recovery risk

---

## Launch Operations Checklist

- migrations run on startup
- bootstrap admin path works
- JWT secret is set securely
- WS auth and origin policy are configured
- logs are structured and reviewable
- backup and restore process is documented
- rollback expectations are documented
