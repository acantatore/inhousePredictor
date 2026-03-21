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

## What Already Exists

The implementation plan should reuse existing design decisions rather than inventing a second product definition.

Existing design assets already locked in:

- `DESIGN.md` navigation model for desktop and mobile
- `DESIGN.md` information architecture for app shell and core screens
- `DESIGN.md` screen blueprints for Markets, Market Detail, Portfolio, Create Market, Resolve Market, Dispute Submission, and Admin Disputes
- `DESIGN.md` component vocabulary including `Market Row`, `Trade Ticket`, `Deadline Block`, `Evidence Card`, `Dispute Banner`, and `Performance Stat`
- `DESIGN.md` interaction state matrix, responsive behavior, and accessibility requirements
- `COPY_GUIDE.md` tone, action labels, error language, and empty-state rules

Implementation should extend these patterns, not replace them.

---

## Not In Scope

The following design work is explicitly not part of this implementation plan:

- public marketing pages, launch pages, or growth-site work
- a generic enterprise admin console
- advanced charting or analytics-first market views
- tournament, leaderboard, or social-feed mechanics
- final brand assets, illustration system, or marketing-style hero art

These are intentionally deferred so the team can focus on a trustworthy internal forecasting product.

---

## Information Architecture Execution Rules

Implementation must preserve the screen hierarchy already established in `DESIGN.md`.

### Markets

Users should notice, in order:

1. what needs attention now
2. what the crowd currently thinks
3. whether they already have a stake

Execution implications:

- `Closing Soon` must appear before general open markets
- probability, deadline, and user-position cues must be scannable in one row
- the team-context explanation should orient first-time users without pushing markets below the fold unnecessarily

### Market Detail

Users should notice, in order:

1. what the market is asking
2. whether action is still available
3. what the current crowd signal is
4. why the eventual outcome can be trusted
5. what the user can do right now

Execution implications:

- trade panel cannot dominate the page before the market question and timing context are clear
- resolver identity, evidence, and dispute state are trust elements, not footer metadata
- disabled trading states must become explanation surfaces, not dead forms

### Portfolio

Users should notice, in order:

1. how they are doing overall
2. which predictions are still unresolved
3. what resolved recently

Execution implications:

- summary stats lead the page
- unresolved holdings are prioritized above historical detail
- copy and visuals must avoid implying sell-based or live-liquid trading mechanics that do not exist in v1

---

## Anti-Generic UI Guardrails

The frontend implementation should not fall back to generic SaaS dashboard patterns just because they are easy to assemble.

Avoid:

- interchangeable card-grid dashboards where all content competes equally
- oversized marketing-style hero treatments inside the product shell
- finance-terminal styling, ticker behavior, or noisy real-time animations
- empty states that read like placeholders
- visual hierarchy that makes controls louder than the market question or trust signals

Prefer:

- structured lists and rows that support fast scanning
- sectioning that emphasizes timing, probability, and trust
- calm, editorial-feeling layouts instead of widget collections
- action areas that feel grounded in explanation, not conversion pressure
- trust surfaces such as resolver identity, evidence, and dispute state treated as core content

Implementation rules:

- the Markets page should feel like a forecasting workspace, not a dashboard of unrelated widgets
- Market Detail should prioritize the market question and timing context before trade mechanics
- Portfolio should feel reflective and performance-oriented, not like a brokerage account
- Admin Disputes should stay narrow and task-focused rather than expanding into a control-center layout
- if a UI pattern is not already supported by `DESIGN.md` component vocabulary, it should justify itself before being introduced

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
- lock dispute state transitions and payout interaction rules before admin UI work begins

### Dependencies

- schema changes for payout tracking and pool cleanup
- agreement on target error contract
- agreement on allowed dispute outcomes and whether payout is deferred, confirmed, or overridden during dispute handling

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
- define bounded market-list query behavior for launch routes so the default Markets screen does not depend on unbounded result sets

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
- add startup and operational-path tests for migrations, bootstrap-admin behavior, and launch-critical runtime checks

### Exit Criteria

- critical business rules are covered by automated tests
- regressions in payout, deadline, or authorization behavior are caught automatically
- startup and launch-path regressions are caught before release instead of during deploy

---

## Phase 4.5 - V1 API Contract Freeze

### Goal

Freeze the user-facing backend contract before real frontend integration begins.

### Required Work

- lock request and response shapes for auth, markets, market detail, trade, positions, resolve, dispute, and admin dispute surfaces
- lock standardized error responses for validation, auth, not-found, conflict, busy, closed, and unauthorized states
- lock creator-trading restriction behavior and message contract
- lock WS authentication and origin requirements for launch-ready environments
- verify frontend-critical contract paths in integration tests

### Exit Criteria

- frontend work no longer depends on temporary backend semantics
- user-facing error and state handling can be implemented once instead of reworked repeatedly
- the API contract documented in `API.md` is the implementation target for Phase 5 onward

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

- frozen v1 API contract for all user-facing surfaces
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

## Interaction State Coverage Required Before Launch

The following states are part of the feature definition, not optional polish.

| Surface | Loading | Empty | Error | Success | Partial / Live |
|---|---|---|---|---|---|
| Auth | disabled submit with inline progress | n/a | invalid credentials, duplicate email, validation guidance | logged in or account created | retained form input after recoverable error |
| Markets | skeleton rows and filters | explain what markets are, CTA to create | failed to load markets | filter or view mode applied | stale prices or disconnected live feed |
| Market Detail | skeleton header and trade panel | n/a | missing market, unauthorized, busy, closed-state explanation | trade placed, dispute filed, resolution shown | content loaded while activity or live data is delayed |
| Portfolio | skeleton stat blocks and rows | no positions yet, CTA to explore markets | failed to load portfolio | n/a | some metrics loaded before detailed rows |
| Create Market | disabled submit while validating | n/a | inline validation and submit failure | market created | local draft-like state preserved after recoverable error |
| Resolve Market | disabled submit | n/a | invalid evidence, unauthorized, invalid market state | resolution recorded | confirmation visible while downstream market data refreshes |
| Dispute Submission | disabled submit | dispute unavailable because market is not eligible | validation failure, deadline closed, unauthorized | dispute submitted with next-step explanation | market page refresh lag after submit |
| Admin Disputes | skeleton queue | no disputes to review | queue failed to load, action failed | dispute action completed | queue data refreshing or stale |

Implementation rules:

- empty states must include warmth, context, and one clear next action
- disabled states must explain why action is unavailable
- partial and live-update states must degrade gracefully without hiding core content
- success states should confirm what changed without sounding like a trading app

Phase ownership:

- Phase 5 must deliver auth, markets, and market-detail loading, empty, and error states
- Phase 6 must deliver trade, portfolio, create, resolve, and dispute state coverage
- Phase 7 must deliver admin dispute queue state coverage
- Phase 8 must verify all states against `QA.md` before launch signoff

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

## Code Quality Guardrails

- backend is the source of truth for business validation, authorization rules, and typed error semantics
- frontend may mirror validation for usability, but must not become the only place a rule exists
- shared response and error helpers should be reused across handlers instead of copied per route
- UI state handling should consume documented error codes and states rather than infer backend behavior ad hoc
- implementation should prefer extending existing market, trade, auth, and ws modules over inventing parallel orchestration layers without a clear boundary

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
- calibration scores
- seasonal tournaments
- market archival
- expanded admin and audit tooling
