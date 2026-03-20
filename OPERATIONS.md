# InhousePredictor Operations Guide

> Last updated: 2026-03-20

---

## Purpose

This document defines how InhousePredictor should be run, bootstrapped, observed, and recovered.

The project is still early, so this guide covers both the current local-development reality and the target operating model required before first users.

---

## Runtime Overview

- API server written in Go 1.22
- PostgreSQL 16 as primary datastore
- WebSocket support for live updates
- Docker and docker-compose for local startup

---

## Local Development

### Current Flow

```bash
cp .env.example .env
# edit .env and set JWT_SECRET
docker-compose up
```

Expected local endpoints:

- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`

### Notes

- current schema application relies on container startup behavior
- frontend runtime does not exist yet

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

- database schema is applied automatically only under the current container setup assumptions

### Target State

- application runs migrations on startup before serving traffic
- startup fails clearly if migrations cannot be applied
- startup optionally bootstraps the first admin when configured

---

## Database And Migrations

### Current State

- schema lives in `migrations/001_init.sql`
- there is no general migration runner in the application

### Required Improvements

- embed migration execution into normal startup
- add future migrations for payout idempotency and pool cleanup
- avoid environment setups that require a fresh container to apply schema changes

---

## Admin Bootstrap

### Current State

- first admin must be created by direct database update

### Target State

- if `BOOTSTRAP_ADMIN_EMAIL` is set and no admin exists, promote that user on startup

### Operational Reason

- first-run setup should not require raw SQL access

---

## Logging And Observability

### Current State

- structured logging is not yet implemented

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

- graceful shutdown is missing

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
- backend deployment is the only runtime concern today
- frontend deployment can be documented after frontend implementation exists

---

## Operational Risks

- migration behavior is too fragile in current form
- manual admin bootstrap is easy to forget and error-prone
- lack of structured logs makes incident analysis difficult
- missing graceful shutdown can create noisy deploy behavior
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
