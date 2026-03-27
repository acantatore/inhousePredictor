# InhousePredictor — Architecture & State of the Project

> Last updated: 2026-03-27

---

## What It Is

InhousePredictor is a **play-money internal prediction market platform** for companies — a Polymarket clone you run inside your org. Employees spend "points" (not real money) to bet on outcomes like OKRs hitting, SLA breaches, hiring decisions, or financial targets. Markets resolve to YES or NO; correct bettors win a proportional share of the total pool.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.24 |
| HTTP Router | [chi/v5](https://github.com/go-chi/chi) |
| Database | PostgreSQL 16 |
| DB Driver | [pgx/v5](https://github.com/jackc/pgx) |
| Auth | JWT (HS256, 24-hour expiry) |
| Real-time | WebSocket via gorilla/websocket |
| Containers | Docker + docker-compose |
| Password Hashing | bcrypt (golang.org/x/crypto) |
| IDs | UUID v4 (google/uuid) |

---

## Directory Structure

```
inhousePredictor/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: routing, DI wiring, server start
├── internal/
│   ├── auth/
│   │   ├── jwt.go               # Token generation & parsing
│   │   └── middleware.go        # JWT extraction middleware, AdminOnly guard
│   ├── cpmm/
│   │   └── cpmm.go              # Constant Product Market Maker algorithm
│   ├── db/
│   │   └── db.go                # pgxpool connection setup
│   ├── httpx/
│   │   └── httpx.go             # Shared JSON success/error helpers
│   ├── market/
│   │   ├── handler.go           # HTTP handlers (Create, Get, List, Resolve, Dispute)
│   │   ├── model.go             # Market & Pool structs
│   │   ├── repository.go        # DB queries for markets
│   │   └── service.go           # Business rules for markets
│   ├── migrate/
│   │   └── migrate.go           # Startup schema reconciliation
│   ├── trade/
│   │   ├── handler.go           # HTTP handlers (Trade, MyPositions, MarketTrades)
│   │   ├── model.go             # Trade & Position structs
│   │   ├── repository.go        # Atomic trade execution queries
│   │   └── service.go           # Trade orchestration + WebSocket broadcast
│   ├── user/
│   │   ├── handler.go           # HTTP handlers (Register, Login, Me)
│   │   ├── model.go             # User struct
│   │   ├── repository.go        # User DB queries
│   │   └── service.go           # Register/Authenticate logic
│   ├── validate/
│   │   └── validate.go          # Input validation helpers
│   └── ws/
│       ├── handler.go           # WebSocket upgrade + read/write pumps
│       └── hub.go               # Pub/sub broadcaster for price updates
├── migrations/
│   └── 001_init.sql             # Full PostgreSQL schema
├── frontend/                    # React + Vite frontend application
├── docker-compose.yml
├── Dockerfile                   # Multi-stage build
├── .env.example
└── go.mod
```

---

## Database Schema

### `users`
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| name | TEXT | |
| email | TEXT | UNIQUE |
| password_hash | TEXT | bcrypt |
| balance | BIGINT | Default 10,000 points |
| is_admin | BOOLEAN | |
| created_at | TIMESTAMPTZ | |

### `markets`
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| question | TEXT | |
| description | TEXT | |
| category | ENUM | `people \| okrs \| slas \| financials \| general` |
| creator_id | UUID FK | users |
| resolver_id | UUID FK | users |
| status | ENUM | `open \| closed \| resolved \| disputed \| cancelled` |
| outcome | ENUM | `yes \| no \| cancelled` |
| initial_liquidity | BIGINT | Min 100 points |
| closes_at | TIMESTAMPTZ | Trades should stop here (enforcement broken — see known issues) |
| resolves_at | TIMESTAMPTZ | |
| resolved_at | TIMESTAMPTZ | |
| dispute_deadline | TIMESTAMPTZ | 48h after resolution |
| payout_at | TIMESTAMPTZ | Set when payout/refund has been finalized |
| evidence_url | TEXT | Set on resolution |

### `pools` (one-to-one with markets)
| Column | Type | Notes |
|---|---|---|
| market_id | UUID PK | |
| yes_reserve | DOUBLE PRECISION | |
| no_reserve | DOUBLE PRECISION | |
| total_collateral | BIGINT | Sum of all cost spent on this market |
| updated_at | TIMESTAMPTZ | |

### `trades` (immutable ledger)
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| user_id, market_id | UUID FK | |
| side | ENUM | `yes \| no \| sell_yes \| sell_no` |
| shares | DOUBLE PRECISION | Shares bought or sold |
| cost | BIGINT | Buy points spent or whole shares requested for sell orders |
| yes_price_before | DOUBLE PRECISION | Implied prob before trade |
| yes_price_after | DOUBLE PRECISION | Implied prob after trade |
| created_at | TIMESTAMPTZ | |

### `positions`
| Column | Type | Notes |
|---|---|---|
| (user_id, market_id) | Composite PK | |
| yes_shares | DOUBLE PRECISION | |
| no_shares | DOUBLE PRECISION | |

### `disputes`
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| market_id, user_id | UUID FK | |
| resolved_by | UUID FK | Nullable |
| reason | TEXT | |
| created_at, resolved_at | TIMESTAMPTZ | |

---

## API Reference

### Public Endpoints
```
POST /auth/register    body: {name, email, password}
POST /auth/login       body: {email, password} → {data: {token, user}}
GET  /ws               Upgrade to WebSocket (optional ?market_id=<uuid>, Bearer auth required)
```

### Authenticated Endpoints (Bearer <JWT> required)
```
GET  /me

GET  /markets                      ?category=<enum>&status=<enum>  (bounded to 50 rows)
POST /markets                      body: {question, description, category, resolver_id, initial_liquidity, closes_at, resolves_at}
GET  /markets/{id}
POST /markets/{id}/resolve         body: {outcome: "yes"|"no", evidence_url}  — resolver only
POST /markets/{id}/dispute         body: {reason}
POST /markets/{id}/review-dispute  body: {action, outcome?, evidence_url?}       — admin only

POST /markets/{id}/trade           body: {side: "yes"|"no"|"sell_yes"|"sell_no", cost: <int64>}  # buy points or whole shares to sell
GET  /markets/{id}/trades          Returns last 50 trades
GET  /positions                    User's holdings across all markets
```

### WebSocket Messages (server → client)
```json
{
  "type": "price_update",
  "market_id": "<uuid>",
  "payload": {
    "yes_price": 0.65,
    "no_price": 0.35,
    "last_trade": { "side": "yes", "shares": 10.5, "cost": 1000 }
  }
}
```

Subscribe to a single market: `GET /ws?market_id=<uuid>`
Subscribe to all markets: `GET /ws` (no query param)

---

## Market Mechanics — CPMM

InhousePredictor uses a **Constant Product Market Maker** (same model as Uniswap v2):

```
yes_reserve × no_reserve = K
```

**Prices** (implied probabilities):
```
YES price = no_reserve  / (yes_reserve + no_reserve)
NO  price = yes_reserve / (yes_reserve + no_reserve)
```

**Buying YES with `cost` points:**
1. Add `cost` to `no_reserve` → new_no = no_reserve + cost
2. Solve for new_yes: `new_yes = K / new_no`
3. Shares received = `yes_reserve - new_yes`

Markets start at **50/50** (`yes_reserve = no_reserve = initial_liquidity`).

**Payout on resolution:**
```
payout_per_share = total_collateral / total_winning_shares
user_payout = user_winning_shares × payout_per_share
```

---

## Trade Execution — Concurrency Model

All trades run at **SERIALIZABLE isolation** with `SELECT ... FOR UPDATE` on the market and pool rows. This prevents:
- Double-spend (balance deducted atomically)
- K-invariant violation from concurrent trades

Serialization failures are retried and then returned as a user-safe `market_busy` error.

---

## Authentication

- Registration creates a user with bcrypt-hashed password and 10,000 starting balance.
- Login returns an HS256 JWT (24-hour expiry) containing `user_id` and `is_admin`.
- `JWT_SECRET` is set via environment variable.
- Admin flag is stamped into the token at issue time — no runtime role change without a new login.

---

## Real-time Architecture

```
Trade executed
     │
     ▼
TradeService.Execute()
     │
     ▼
ws.Hub.Broadcast(message)
     │
     ├── Clients subscribed to this market_id → message sent
     └── Clients subscribed to all markets ("") → message sent

Slow clients: dropped (non-blocking channel send). No persistence.
```

---

## Running Locally

```bash
cp .env.example .env
# Edit .env: set JWT_SECRET to a random string

docker-compose up
# API: http://localhost:8080
# DB:  localhost:5432
```

Schema is reconciled on application startup.

First admin bootstrap is supported with `BOOTSTRAP_ADMIN_EMAIL`.

---

## Known Issues & Roadmap

### P1 — Must fix before first users

| # | Issue | Location |
|---|---|---|
| 1 | Resolved markets finalize payouts lazily on later app activity, not by background worker | `trade/payout.go`, `market/service.go` |
| 2 | Cancelled-market disputes do not yet have a dedicated user-facing explanation path | `market/handler.go` |
| 3 | No API-level pagination cursor yet; list is only bounded to 50 rows | `market/repository.go` |

### P2 — Before general availability

- [x] **Sell/exit mechanism** — ✅ IMPLEMENTED. Users can now sell shares back to the pool before market resolution.
- [x] **Structured logging (`slog`)** — ✅ IMPLEMENTED
- Full cursor-based market list pagination beyond bounded launch query behavior
- [x] **Creator trading restriction enforcement** — ✅ IMPLEMENTED (policy locked: creators cannot trade their own markets)

### P3 — Nice to have

- K-invariant drift alerting
- Market archival for resolved markets > 1 year old
- Per-user Brier score (forecast calibration metric)
- Seasonal tournaments with point resets
- CORS lockdown (currently accepts any WebSocket origin)
- Standardized error response format

---

## What Doesn't Exist Yet

- **Admin panel** — No broader management surface beyond the dispute workflow
- **Rate limiting** — No request throttling
- **CI/CD** — No pipeline or linting config
