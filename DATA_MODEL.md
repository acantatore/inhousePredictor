# InhousePredictor Data Model

> Last updated: 2026-03-20

---

## Purpose

This document describes the canonical product data model for InhousePredictor.
It translates the current schema into domain concepts, ownership rules, invariants, and planned schema changes.

---

## Domain Overview

InhousePredictor models a company forecasting system with six core entities:

- `User`
- `Market`
- `Pool`
- `Trade`
- `Position`
- `Dispute`

Together these represent account identity, market definition, pricing state, immutable trade history, user holdings, and governance workflows.

---

## Entity Definitions

### User

Represents an authenticated employee account.

Key fields:

- `id` - UUID primary key
- `name`
- `email` - unique
- `password_hash`
- `balance` - integer points, defaults to `10000`
- `is_admin`
- `created_at`

Rules:

- balances are internal points only
- email must be unique
- password is stored only as a bcrypt hash
- admin status affects authorization, not user identity

### Market

Represents a binary forecasting question.

Key fields:

- `id`
- `question`
- `description`
- `category`
- `creator_id`
- `resolver_id`
- `status`
- `outcome`
- `initial_liquidity`
- `closes_at`
- `resolves_at`
- `resolved_at`
- `dispute_deadline`
- `evidence_url`

Rules:

- one creator and one resolver are assigned per market
- creator is not allowed to trade the market
- resolver is responsible for resolution
- evidence is required for resolution
- `closes_at` must be before `resolves_at`

### Pool

Represents the pricing state for one market.

Key fields:

- `market_id` - primary key and foreign key
- `yes_reserve`
- `no_reserve`
- `total_collateral`
- `updated_at`

Rules:

- exactly one pool exists per market
- market prices are derived from pool reserves
- total collateral is the total points spent into the market
- stored `k` should be removed; invariant should be computed inline

### Trade

Represents one executed buy event.

Key fields:

- `id`
- `user_id`
- `market_id`
- `side`
- `shares`
- `cost`
- `yes_price_before`
- `yes_price_after`
- `created_at`

Rules:

- trade ledger is immutable
- trade `cost` is integer points spent
- `shares` can be fractional
- trades should only occur while market trading is open

### Position

Represents the user's current share holdings in a market.

Key fields:

- composite primary key: `(user_id, market_id)`
- `yes_shares`
- `no_shares`

Rules:

- at most one position row exists per user-market pair
- positions are derived and updated from the trade ledger
- current v1 model is buy-only, so shares only increase before resolution

### Dispute

Represents a challenge to a market resolution.

Key fields:

- `id`
- `market_id`
- `user_id`
- `resolved_by`
- `reason`
- `created_at`
- `resolved_at`

Rules:

- disputes should only be created during the eligible dispute window
- admin workflow should eventually complete each dispute lifecycle in-product

---

## Enums

### Market Category

- `people`
- `okrs`
- `slas`
- `financials`
- `general`

### Market Status

- `open`
- `closed`
- `resolved`
- `disputed`
- `cancelled`

### Market Outcome

- `yes`
- `no`
- `cancelled`

### Trade Side

- `yes`
- `no`

---

## Relationships

- one `User` can create many `Market` records
- one `User` can resolve many `Market` records
- one `Market` has exactly one `Pool`
- one `Market` has many `Trade` records
- one `User` has many `Trade` records
- one `User` has at most one `Position` per `Market`
- one `Market` can have many `Dispute` records over time if policy allows, though v1 should keep this flow simple

---

## Derived Values

### Price

Prices are derived from reserve state.

```text
YES price = no_reserve / (yes_reserve + no_reserve)
NO price  = yes_reserve / (yes_reserve + no_reserve)
```

### CPMM Invariant

```text
K = yes_reserve * no_reserve
```

This should be computed from reserves, not stored as mutable state.

### Payout

```text
payout_per_share = total_collateral / total_winning_shares
user_payout = user_winning_shares * payout_per_share
```

Payout processing must be idempotent.

---

## Lifecycle Rules

### User Lifecycle

- user registers
- user authenticates
- user receives starting balance
- user participates as employee, resolver, and possibly admin depending on role

### Market Lifecycle

- market created with initial liquidity and 50/50 start
- market remains open for trading until `closes_at`
- market becomes non-tradable after close
- resolver records outcome with evidence
- dispute window opens according to policy
- market may move into disputed handling if challenged
- payout is applied safely

### Trade Lifecycle

- request validated
- authorization and market state checked
- trade executed transactionally
- ledger written
- position updated
- pool updated
- WS broadcast sent

### Dispute Lifecycle

- user files dispute with reason
- dispute is visible in admin queue
- admin reviews and resolves according to the final dispute workflow

---

## Data Invariants

- one pool per market
- one position row per user-market pair
- user email is unique
- user balance cannot go negative
- creator cannot trade own market
- trades cannot execute after `closes_at`
- disputes cannot be created after `dispute_deadline`
- payout cannot apply twice
- market timing should satisfy `closes_at < resolves_at`
- normal user views should expose aggregated market activity, not named public trade history

---

## Planned Schema Changes

### Add `markets.payout_at`

Purpose:

- make payout processing idempotent

### Remove `pools.k`

Purpose:

- avoid drift caused by stored float state
- compute invariant from reserves when needed

---

## Data Quality And Validation Expectations

- question length should be capped
- description length should be capped
- passwords should enforce a minimum length
- `evidence_url` should be validated as a URL
- timing fields should be validated at creation time

---

## Notes For Future Expansion

- sell and exit mechanics will require additional trade semantics and position updates
- audit-oriented admin workflows may require more event history or actor metadata
- pagination may introduce cursor fields or stable sort contracts on list endpoints
