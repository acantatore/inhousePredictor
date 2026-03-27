# InhousePredictor QA Plan

> Last updated: 2026-03-27

---

## Purpose

This document defines the test strategy, release gates, and critical acceptance criteria for InhousePredictor v1.

Because the system handles balances, market outcomes, and trust-sensitive governance actions, QA must focus first on correctness, authorization, and operational safety.

---

## QA Principles

- trust-sensitive bugs are release blockers
- market logic must be tested at the unit and integration level
- backend policy rules must be enforced server-side
- UI should explain failures clearly instead of hiding them
- launch quality depends on both automated and end-to-end validation

---

## Test Layers

### Unit Tests

Primary target:

- `internal/cpmm/cpmm.go`

Required coverage:

- K invariant behavior after buy YES and buy NO
- price symmetry
- price bounds remain valid
- payout math sums to total collateral
- slippage behavior is real and non-reversible

### Integration Tests

Primary target:

- service and repository layers against real PostgreSQL using testcontainers-go

Required coverage:

- registration and login
- duplicate email rejection
- market creation and initial liquidity debit
- trade execution atomicity
- insufficient balance rejection
- creator self-trading rejection
- close-time enforcement
- resolution with evidence
- payout idempotency
- dispute submission and deadline enforcement
- admin dispute handling once implemented

### API Contract Tests

Primary target:

- HTTP handlers and response shapes

Required coverage:

- auth failure shape
- validation failure shape
- busy market shape
- not found shape
- conflict shape for duplicate email

### WebSocket Tests

Required coverage:

- authenticated handshake
- origin allowlist enforcement
- all-market subscription behavior
- market-scoped subscription behavior
- price update delivery after trades
- behavior when clients disconnect or become slow

### Frontend End-To-End Tests

Required coverage once UI exists:

- register or log in
- browse markets
- open market detail
- buy YES or NO
- sell an existing position
- create market
- resolve market as resolver
- submit dispute
- review positions in portfolio
- review disputes as admin

---

## Critical Launch Scenarios

These are mandatory pass scenarios before first internal users.

### Trading Safety

- trades after `closes_at` are rejected
- creator cannot trade own market
- insufficient balance blocks trade cleanly
- concurrent trades do not corrupt balances or pool state
- serialization conflicts retry safely and return a user-safe busy response when retries are exhausted

### Resolution Safety

- only the assigned resolver can resolve a market
- evidence URL is required for resolution
- payout cannot apply twice
- outcome and evidence remain visible after resolution

### Dispute Safety

- disputes are rejected after the deadline
- dispute submission records actor and reason correctly
- admin-only dispute actions are enforced once implemented

### Security And Auth

- protected routes reject missing or invalid JWTs
- admin-only and resolver-only routes enforce authorization correctly
- WebSocket requires auth and approved origin before launch

### User Experience Baseline

- empty states include context and next action
- disabled actions explain why they are unavailable
- success and error feedback is visible in the relevant flow

---

## Acceptance Criteria By Feature

### Auth

- user can register with valid input
- duplicate email returns safe conflict response
- user can log in and receive JWT
- invalid credentials return a safe auth error

### Markets

- user can list and filter markets
- user can create a market with valid inputs
- invalid timing is rejected
- initial liquidity is deducted from creator balance exactly once

### Trading

- user can buy YES or NO with valid cost input
- user can sell an existing YES or NO position with a whole-share sell input
- estimated shares align with executed shares within expected contract behavior
- user balance updates correctly
- WS price update is emitted after successful trade

### Resolution

- assigned resolver can resolve with evidence URL
- non-resolver cannot resolve
- payout processing is idempotent

### Disputes

- eligible user can file dispute with reason
- dispute deadline is enforced
- admin workflow functions end to end once implemented

### Portfolio

- user sees current holdings and resolved outcomes
- portfolio and trade surfaces explain sell support without drifting into trader-style live PnL language

---

## Regression Checklist

- duplicate email no longer leaks raw Postgres error
- trade after close is blocked
- dispute after deadline is blocked
- payout repeat attempt does not duplicate balance changes
- raw serialization error is not shown to the user
- creator self-trading is blocked
- WS unauthenticated or invalid-origin attempts are rejected

---

## Release Checklist

- CPMM unit tests exist and pass
- integration tests exist and pass
- critical auth and role checks are covered
- launch blockers from `TODOS.md` are closed
- admin dispute flow exists and is tested
- WS auth and origin restrictions are verified
- API error shapes are consistent in core flows
- local startup and migration flow are documented and verified

---

## Manual QA Focus Areas

- first-time onboarding clarity
- market scan speed on desktop and mobile
- trust cues in market detail
- error clarity during trade failures
- dispute and resolution UX tone
- empty states across markets and portfolio

---

## Non-Goals For QA v1

- advanced performance benchmarking
- cosmetic charting QA for features not in scope
- public marketing site QA
