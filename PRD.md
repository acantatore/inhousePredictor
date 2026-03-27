# InhousePredictor Product Requirements Document

> Last updated: 2026-03-27

---

## Product Summary

InhousePredictor is a private, play-money prediction market platform for companies.
Employees use internal points to forecast binary outcomes such as OKR delivery, SLA misses, hiring outcomes, financial targets, and other company-relevant questions.

The product is designed to help teams surface collective judgment, make uncertainty visible, and encourage better decision-making without creating a trading-first or gambling-like experience.

---

## Problem

Companies often have fragmented, informal opinions about future outcomes, but no lightweight system for turning those opinions into visible, testable forecasts.

Today, teams rely on:

- opinionated Slack threads
- top-down status updates
- vague confidence statements
- unstructured debate without accountability

These patterns make it hard to see what the organization actually believes, what is changing over time, and where confidence is weak or overconfident.

InhousePredictor creates a shared forecasting layer that helps employees:

- make uncertainty explicit
- see the current crowd signal
- participate safely using play-money points
- review outcomes after resolution
- build better judgment over time

---

## Product Thesis

This should feel like a calm internal forecasting workspace for normal employees, not a trader terminal.

The product wins if employees can quickly understand:

- what the company is predicting
- what the crowd currently thinks
- which markets need attention now
- what action they can take next
- why resolved outcomes can be trusted

---

## Users

### Employee

Primary participant.

Responsibilities and needs:

- browse and understand active markets
- buy YES or NO with internal points
- create markets
- track open positions and resolved outcomes
- file disputes when resolution appears incorrect

### Resolver

Assigned decision-maker for a specific market.

Responsibilities and needs:

- resolve assigned markets
- attach evidence URL at resolution time
- understand dispute-window expectations
- provide a lightweight and trustworthy resolution flow

### Admin

Operational backstop for disputed markets.

Responsibilities and needs:

- review disputed markets
- handle resolution edge cases
- work from a focused dispute queue rather than a broad admin suite in v1

---

## Core Jobs To Be Done

### For Employees

- When I want to understand what the company is uncertain about, I can scan current markets quickly.
- When I have an opinion, I can express it by spending points on YES or NO.
- When I want to forecast a new question, I can create a market without admin involvement.
- When I want to understand how I am doing, I can review my positions and resolved outcomes.
- When I believe a market was resolved incorrectly, I can file a dispute.

### For Resolvers

- When a market reaches resolution, I can resolve it clearly and attach evidence.

### For Admins

- When a dispute is filed, I can review it from a dedicated queue and take the appropriate dispute action.

---

## v1 Scope

### In Scope

- registration and login
- authenticated app shell
- markets home
- filters by category and status
- market detail
- binary YES/NO trading with buy and sell support using play-money points
- real-time price updates through WebSocket
- user portfolio and position history
- create market flow
- resolver-only market resolution flow
- dispute submission flow
- basic admin dispute queue and workflow

### In Scope But Must Be Fixed Before First Users

- enforce `closes_at` in trade path
- make payout idempotent
- enforce dispute deadline server-side
- handle serialization failures safely
- remove stored `k` drift risk
- standardize duplicate email error handling
- add migration runner on startup
- add first-admin bootstrap path
- require authenticated and origin-restricted WebSocket access
- add shared JSON error/response layer
- add unit and integration tests

### Out Of Scope For v1

- public marketing site
- full admin management console
- advanced charts
- tournaments or seasonal resets
- calibration leaderboards
- social feed features
- expert-mode trader UI
- public anonymous access

---

## Governance And Trust Rules

- The product is play-money only.
- Any authenticated employee can create markets.
- Creators cannot trade their own markets.
- Resolver identity is always visible.
- Evidence is required for resolution.
- Normal users should see aggregated activity, not a named public trade tape.
- Disputes must be available during the eligible dispute window.
- Deadlines must be visible and understandable, not hidden in low-priority metadata.

---

## Functional Requirements

### Authentication

- Users can register with name, email, and password.
- Users can log in and receive a bearer token.
- Authenticated users can access their profile and product surfaces.

### Markets

- Users can list markets with category and status filters.
- Users can open a market detail page.
- Users can create a market with question, context, resolver, timing, and initial liquidity.
- Markets have visible close and resolve times.

### Trading

- Users can buy YES or NO by entering points to spend.
- Users can sell existing YES or NO positions by entering whole shares to sell.
- Trading must stop after `closes_at`.
- Users cannot trade their own markets.
- Users cannot trade without sufficient balance.
- Users receive clear confirmation or actionable errors.

### Resolution And Disputes

- Assigned resolvers can resolve markets with outcome and evidence URL.
- Users can file disputes while eligible.
- Admins can review disputes through a focused queue.

### Portfolio

- Users can review current positions and resolved outcomes.
- Portfolio should show committed points and resolved results.
- Portfolio should show positions and results clearly without drifting into trader-style live PnL framing.

### Real-Time Updates

- Users receive price updates and recent trade context through WebSocket subscriptions.
- Live updates should feel current but not noisy.

---

## Non-Functional Requirements

- API responses should use a consistent JSON contract.
- Trade execution must be safe under concurrent load.
- Security boundaries must match an internal company product.
- Core flows should work on desktop and mobile.
- Empty, loading, success, and error states must be intentional.
- Launch readiness requires unit and integration test coverage for critical flows.

---

## Success Metrics

### Product Adoption

- at least one company can run internal forecasting without manual database intervention
- employees can create, trade, resolve, and dispute markets using only documented product flows

### Trust And Reliability

- no successful trades after market close
- no duplicate payout credits from repeated resolution attempts
- no disputes accepted after dispute deadline
- no raw database errors exposed to end users in normal flows

### UX Quality

- first-time users can understand how to participate without knowing prediction market jargon
- disabled actions explain what is happening and what to do next
- resolution evidence and resolver identity are easy to find

### Engineering Readiness

- CPMM unit tests cover pricing and payout invariants
- integration tests cover market creation, trading, payout, and authorization-critical flows
- WebSocket auth and origin restrictions are enforced before launch

---

## Launch Criteria

The product is ready for first internal users only when all of the following are true:

- P1 launch blockers from `TODOS.md` are resolved
- admin dispute workflow exists end to end
- shared JSON error handling is in place
- WebSocket access is authenticated and origin-restricted
- migrations run reliably on startup
- first admin can be bootstrapped without raw manual SQL
- backend tests cover trading, payout, and deadline enforcement
- core frontend flows exist for auth, markets, trade, create, resolve, dispute, and portfolio

---

## Assumptions

- the product is used inside a company trust boundary
- companies are comfortable with internal point-based incentives
- market categories in v1 are broad enough for early usage
- one resolver per market is sufficient for v1
- a focused admin dispute queue is enough for early governance needs

---

## Risks

- if trust-sensitive rules are not enforced server-side, user confidence will drop quickly
- if WS remains unauthenticated or unrestricted, the product violates its internal-only posture
- if payout and deadline bugs persist, the market system becomes unreliable
- if frontend copy drifts into trading jargon, the product will feel more intimidating than intended
- if market creation is too loose, question quality may degrade

---

## Open Questions

- what exact dispute outcomes can admins apply after review?
- what final JSON error envelope should all handlers use?
- should sell orders stay whole-share only, or should the API contract evolve to support fractional sells?
- what pagination contract should the market list use before broader rollout?
