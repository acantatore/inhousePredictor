# InhousePredictor Implementation Plan

> Last updated: 2026-03-20

---

## Goal

Ship a launch-ready internal prediction market product that is safe, trustworthy, and usable by normal employees.

This plan assumes the current backend exists, the frontend does not yet exist, and several backend issues identified in `ARCHITECTURE.md` and `TODOS.md` must be fixed before first users.

---

## Planning Principles

- trust and correctness before polish
- enforce governance rules in the backend, not just the UI
- stabilize the API contract before building most frontend flows
- treat launch blockers as real blockers, not cleanup
- use documentation, tests, and operations work as part of product delivery

---

## Current State

- backend API exists for auth, markets, trading, positions, resolution, and disputes
- WebSocket price updates exist
- frontend directory exists but has no implementation
- several must-fix backend issues remain unresolved
- no unit tests or integration tests exist yet
- no CI/CD pipeline exists yet

---

## Phase 0 - Product Contract Lock

### Goal

Lock the product and engineering contract before major implementation begins.

### Work

- finalize product, design, and architecture docs
- create operational docs for API, data model, QA, security, and operations
- lock creator trading policy as not allowed
- define target response and error conventions for API work

### Exit Criteria

- required planning docs exist
- major trust and policy decisions are written down
- frontend and backend teams can build from the same contract

---

## Phase 1 - Backend Launch Blockers

### Goal

Remove correctness and trust issues that would break the product for first users.

### Required Work

- enforce `closes_at` in the trade execution path
- make payout idempotent
- enforce dispute deadline server-side
- detect serialization failures, retry safely, then return a clean busy response
- remove stored `k` and compute invariant inline
- map duplicate email to a safe conflict response
- add shared JSON error/response layer

### Dependencies

- schema changes for payout tracking and pool cleanup
- agreement on target error contract

### Exit Criteria

- critical trading, payout, and dispute bugs are fixed
- API no longer leaks raw DB errors in core flows
- backend behavior matches documented policy rules

---

## Phase 2 - Platform And Operational Readiness

### Goal

Make the backend operable as a normal application, not a fragile local-only setup.

### Required Work

- embed and run migrations on startup
- add `BOOTSTRAP_ADMIN_EMAIL` flow for first admin creation
- add graceful shutdown handling for HTTP and WebSocket traffic
- add structured logging with `slog`
- document environment variables and runtime expectations

### Dependencies

- migration runner choice
- startup configuration contract

### Exit Criteria

- the app can boot consistently in a new environment
- first admin creation does not require raw DB edits
- logs are useful for launch debugging and incident handling

---

## Phase 3 - Auth, WebSocket, And Security Hardening

### Goal

Bring security boundaries in line with an internal company product.

### Required Work

- require authentication for WebSocket connections
- enforce an approved origin allowlist for WebSocket upgrades
- verify resolver-only and admin-only server-side authorization paths
- tighten validation for passwords, URLs, question length, and timing rules

### Dependencies

- shared auth middleware conventions
- shared error contract

### Exit Criteria

- WS access matches documented auth boundary
- validation failures are explicit and safe
- trust-sensitive authorization rules are enforced consistently

---

## Phase 4 - Backend Test Coverage

### Goal

Create the minimum automated confidence required for launch.

### Required Work

- add CPMM unit tests
- add integration tests with real Postgres via testcontainers-go
- cover market creation, trade execution, payout behavior, deadline enforcement, and auth rules
- add concurrency coverage for serialization retry behavior
- add WS handshake and subscription tests

### Exit Criteria

- critical business rules are covered by automated tests
- regressions in payout, deadline, or authorization behavior are caught automatically

---

## Phase 5 - Frontend Foundation

### Goal

Create the first usable product shell aligned with `DESIGN.md`.

### Required Work

- build app shell and navigation model
- implement login and registration flows
- build Markets home with `Closing Soon`, `Open`, and `Recently Resolved`
- build market list filters and row modes
- implement market detail layout and trade panel

### Dependencies

- stable auth and markets APIs
- stable error responses
- documented copy and flow expectations

### Exit Criteria

- a user can authenticate, browse markets, and inspect a market detail page
- the UI reflects the intended warm, trustworthy product tone

---

## Phase 6 - Core User Flows

### Goal

Deliver the full employee and resolver experience for v1.

### Required Work

- implement buy YES / buy NO flow
- implement portfolio surfaces
- implement create market flow
- implement resolver-only resolution flow
- implement dispute submission flow
- handle all loading, empty, disabled, success, and error states intentionally

### Dependencies

- backend launch blockers fixed
- design and copy rules locked

### Exit Criteria

- a normal employee can complete every core v1 flow without manual support
- a resolver can resolve with evidence
- a user can file a dispute during the valid window

---

## Phase 7 - Admin Dispute Workflow

### Goal

Add the focused admin capability needed to operate the product responsibly.

### Required Work

- define dispute queue API and UI
- implement admin review actions
- persist dispute review actor and timestamps
- document final dispute lifecycle once implementation is locked
- add integration coverage for the dispute workflow

### Dependencies

- payout idempotency fix
- shared error contract
- auth and role enforcement

### Exit Criteria

- disputed markets can be reviewed and completed inside the product
- admins no longer need manual DB work for dispute handling

---

## Phase 8 - Launch Readiness

### Goal

Confirm the system is ready for first internal users.

### Required Work

- run the release checklist from `QA.md`
- verify operations runbook steps
- verify bootstrap admin path
- verify authenticated WS behavior in a realistic environment
- review docs for implementation drift

### Exit Criteria

- release gates pass
- no unresolved P1 items remain
- core docs match shipped behavior

---

## Milestones

### Milestone A - Safe backend core

Backend trust and correctness blockers are closed.

### Milestone B - Operable platform

Migrations, admin bootstrap, logging, and shutdown are in place.

### Milestone C - Usable product shell

Users can authenticate, browse markets, and open market detail.

### Milestone D - End-to-end v1 flows

Trade, portfolio, create, resolve, and dispute flows all work.

### Milestone E - Governed launch

Admin dispute handling, security hardening, QA, and operations checks are complete.

---

## Dependencies And Sequencing

- frontend work depends on a stable API contract and shared error model
- dispute workflow depends on payout safety and authorization rules
- launch depends on WS hardening, not just HTTP auth
- QA automation depends on backend behavior stabilizing first
- operations readiness depends on migration and bootstrap changes landing before deployment

---

## Major Risks

- backend bugs may leak into the first frontend if Phase 1 is rushed
- open dispute lifecycle questions may delay admin flow implementation
- lack of validation could create poor data quality or broken UX
- unauthenticated WS access could undermine the internal-only product posture
- no tests means regressions in core market logic would be hard to detect

---

## Nice-To-Have Work After Launch Readiness

- sell or exit flow
- market list pagination
- calibration scores
- seasonal tournaments
- market archival
- expanded admin and audit tooling
