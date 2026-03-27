# InhousePredictor Launch Readiness Plan

> Last updated: 2026-03-27

---

## Goal

Ship InhousePredictor to first internal users as a trustworthy, operable, and supportable prediction market product.

This is no longer a greenfield build plan. The product already has a working backend, a shipped frontend foundation, buy and sell support, disputes, tests, and release docs. The remaining work is about turning a real but uneven system into something the team can safely launch, monitor, and debug.

---

## Current System Snapshot

Already shipped:

- auth, markets, trading, positions, resolution, disputes, and admin dispute review APIs
- React frontend with market browsing, market detail, trade ticket, portfolio, and create market flows
- buy and sell support, with sell requests currently constrained to whole-share amounts
- authenticated, origin-checked WebSocket updates
- startup migrations, bootstrap-admin support, structured logging, and graceful shutdown
- unit and integration coverage for core backend behavior
- release-management basics: `VERSION`, `CHANGELOG.md`, `CONTRIBUTING.md`

Still launch-critical:

- route and API namespace cleanup so production routing is unambiguous
- explicit trust contracts for trading, payout, resolution, and disputes
- day-1 observability and runbook coverage for trust-sensitive flows
- deployment and rollback rules strong enough for a real first-user launch
- release gates with proof, not just intentions

---

## What This Plan Is Optimizing For

- trust before feature count
- launch clarity before roadmap breadth
- explicit contracts before more frontend polish
- observability before optimism
- rollback posture before launch confidence

---

## Not In Scope For This Launch Plan

- public marketing site or growth-site work
- generic enterprise admin expansion beyond the focused dispute workflow
- advanced charting or analytics-heavy views
- tournaments, leaderboards, or social-feed mechanics
- fractional-share sell support
- final brand system or marketing art direction

These remain valid future bets, but they do not make first-user launch safer.

---

## Launch Definition

InhousePredictor is ready for first internal users only when all of the following are true:

- trust-sensitive product rules are explicitly documented and match code behavior
- core user flows have named, tested failure behavior rather than fallback ambiguity
- the team can detect, investigate, and respond to failures in auth, trading, payout, disputes, startup, and WebSocket delivery
- deployment and rollback steps are documented and realistic
- release gates have owners and proof artifacts

If any one of those is missing, the product may still be demo-ready, but it is not launch-ready.

---

## Launch Control

This section exists to answer one question: how does the team know it is safe to ship?

### Release Gates

| Gate | What must be true | Owner | Proof |
|---|---|---|---|
| Access gate | Auth and market creation behavior are frozen for launch, including validation, authorization, and safe failure behavior | Eng | `API.md` + integration coverage |
| Product trust gate | Trading, sell, resolve, dispute, and payout behavior match documented rules | Product + Eng | updated docs + reviewed state machines |
| API contract gate | All launch-critical endpoints are frozen for launch: auth, markets, trade, positions, resolve, dispute, admin dispute review, and WebSocket auth/origin behavior | Eng | `API.md` + integration coverage |
| UX state gate | Core screens handle loading, empty, disabled, success, partial, and failure states intentionally, including portfolio valuation language and sell-state messaging | Frontend | QA checklist + manual verification |
| Operations gate | Logs, metrics, alerts, and runbooks exist for launch-critical flows | Eng + Ops | ops matrix + alert list + runbook references |
| Deployment gate | Production deployment and rollback steps are documented and tested in a realistic environment | Eng | release checklist + rollback walkthrough |

### Terms Used In This Plan

- `launch-critical` means any flow that can block access, change balances, change market state, affect payouts, or undermine user trust if it fails silently
- `partial state` means the primary screen loads but one supporting surface, live update, or secondary summary is delayed or missing
- `realistic environment` means a deployment path close enough to production to verify routing, auth, startup, WebSocket behavior, logging, and rollback steps without hand-waving

### Ship / No-Ship Criteria

Ship only if:

- no unresolved P1 items remain in `TODOS.md`
- no trust-sensitive flow has undocumented or contradictory behavior
- no launch-critical route depends on backward-compatibility hacks that are not intentional
- no launch-critical failure path is invisible to operators

Do not ship if:

- trading, payout, or dispute behavior depends on “we think this is fine” rather than test coverage or explicit manual verification
- rollback requires improvisation
- the team cannot explain what happens on a failed trade, failed resolution, failed dispute review, or startup migration failure

### Final Ship Authority

If gate owners disagree, final ship authority belongs to the engineering lead and product owner together.

Rules:

- either can block launch if a launch gate is not met
- disagreement defaults to no-ship until the blocking gate is resolved or explicitly waived in writing
- any waived gate must be recorded in the release checklist with owner, reason, and follow-up action

### Required Pre-Launch Proof

- `go test ./...`
- frontend production build succeeds
- manual verification of auth, trade, sell, resolve, dispute, admin dispute review, and portfolio flows
- launch checklist in `QA.md` completed against the latest branch state

### Frontend Launch Baseline

Frontend is launch-ready only if all of the following are true:

- production build succeeds
- launch-critical screens pass the manual QA checklist in `QA.md`
- no debug-only logging or placeholder messaging remains in user-facing flows
- disabled, empty, partial, stale, and retry states are verified for auth, markets, trade, sell, portfolio, create, resolve, and dispute flows

### Rollback Standard

For every launch candidate, the team must be able to answer:

- what code change would be reverted first?
- what migration or data change is irreversible?
- what user-facing behavior would remain degraded even after a code rollback?
- what communication is needed if disputes, payouts, or balances are affected?

---

## Trust Contracts

The product does not earn trust by having code. It earns trust by having explicit, stable rules that backend, frontend, QA, and docs all agree on.

### Trading Contract

```text
OPEN MARKET
  |
  | buy / sell request
  v
VALIDATE REQUEST
  |- invalid input ----------> validation error
  |- creator restricted -----> forbidden / conflict response
  |- insufficient funds -----> insufficient balance error
  |- insufficient shares ----> validation / conflict response
  |- market closed ----------> market_closed error
  |- market busy ------------> market_busy error
  v
EXECUTE TRADE
  |- success ----------------> balance + position + pool update + WS broadcast
  |- serialization failure --> retry path, then market_busy
  |- unexpected failure -----> internal_error + operator-visible logs
```

Launch contract:

- buy requests are spend-first
- sell requests are whole-share only under the current API contract
- user-visible disabled states must match server-enforced restrictions
- no frontend rule may exist without matching backend enforcement

### Resolution / Dispute / Payout Contract

```text
OPEN
  |
  | resolve by assigned resolver
  v
RESOLVED
  |- evidence missing --------> invalid request
  |- resolver unauthorized ---> forbidden
  |- payout pending ----------> payout path must remain idempotent
  v
DISPUTE WINDOW ACTIVE
  |- dispute submitted -------> DISPUTED
  |- deadline passes ---------> payout finality remains valid
  v
DISPUTED
  |- admin confirms outcome --> payout finalized once
  |- admin overrides outcome -> payout finalized once
  |- admin cancels market ----> refund / cancellation contract applies
```

Launch contract:

- every dispute action must produce a named terminal state
- payout must remain idempotent under repeated triggers
- user-visible dispute state must explain what happens next
- cancelled and disputed edge cases cannot rely on operator tribal knowledge

### Contract Freeze Points

Before launch, freeze these behaviors explicitly in docs and tests:

- request / response shapes for auth, markets, trade, positions, resolve, dispute, admin dispute review, and launch-ready WebSocket access
- auth validation and login failure behavior
- market creation validation, liquidity debit rules, and creator restriction behavior
- sell semantics and whole-share constraint
- standardized error shapes for validation, auth, not-found, conflict, busy, closed, and unauthorized states
- dispute review actions and terminal outcomes
- portfolio presentation rules that avoid trader-style live-PnL framing

---

## Day-1 Operations

If something breaks after launch, the first question should not be “where do we even look?”

### Observability Matrix

| Flow | Must log | Must measure | Must alert on |
|---|---|---|---|
| Auth | login failure reason class, registration failure class | auth failure rate | sustained auth spike or login outage |
| Trade / Sell | market id, user id, side, request outcome, failure class | trade success vs failure rate, busy rate | repeated trade failures, elevated busy rate |
| Resolution | resolver id, market id, outcome path, failure class | resolution success/failure | repeated resolution failures |
| Dispute review | admin actor, market id, action, result | dispute action count, failure count | failed admin dispute actions |
| Payout | market id, trigger, idempotent skip vs execute | payout executions, payout skips, payout failures | payout failure |
| Startup | migration result, bootstrap-admin result, listen result | startup success/failure | migration or boot failure |
| WebSocket | auth reject, origin reject, subscription count, drop count | connection count, rejection count, client drop count | auth/origin rejection anomalies or broadcast failures |

### Minimum Runbooks

Before launch, write operator responses for:

- migration failure on startup
- trade path returning unexpected internal errors
- elevated serialization / busy errors
- payout failure or duplicate-trigger concern
- dispute action failure
- WebSocket auth or origin failures after deploy

### Required Operations Artifacts

- an ops matrix stored in `OPERATIONS.md` or a linked runbook doc
- an alert list covering auth, trade, payout, dispute, startup, and WebSocket failures
- at least one deploy/rollback rehearsal record tied to the current release candidate

### Minimum Viable Ops Package

Before launch, the team must have at least:

- logs for every launch-critical flow
- one dashboard covering auth, trade, payout, dispute, startup, and WebSocket health
- one alert per launch-critical flow family
- six runbooks: auth, trade/sell, payout, dispute review, startup/migrations, WebSocket/auth failures
- one completed rehearsal record for deploy + verify + rollback

Alerting baseline:

- one failure alert per launch-critical flow family
- one startup alert for boot or migration failure
- one anomaly alert for elevated trade busy / failure rates

### Day-1 Dashboard Questions

The first dashboard set should let the team answer:

- are users successfully logging in?
- are trades and sells succeeding at normal rates?
- are busy/conflict errors spiking?
- are dispute actions completing?
- did the app start cleanly after the last deploy?

### Debuggability Standard

Three weeks after launch, an engineer should be able to reconstruct a failed trade, failed dispute action, or failed startup from logs and metrics alone.

---

## Remaining Workstreams

### Workstream 1 - Routing And Contract Cleanup

DRI:

- Eng lead

Delivery target:

- before launch rehearsal

Goal:

- make production routing and API boundaries explicit rather than transitional

Dependency note:

- routing and API boundary decisions are prerequisite input to the final contract freeze

Required work:

- publish a routing matrix that shows supported root routes, `/api` routes, frontend routes, and proxy assumptions
- decide whether root-route compatibility stays or is removed before launch, then document the decision in `API.md` and `OPERATIONS.md`
- remove route ambiguity from the release checklist

Exit criteria:

- the team can explain exactly which routes are public, authenticated, proxied, and production-supported

### Workstream 2 - Trust-Sensitive Rule Lock

DRI:

- Product + backend lead

Delivery target:

- before API contract freeze signoff

Goal:

- turn implied trading, payout, and dispute rules into explicit launch contracts

Required work:

- add final dispute lifecycle and terminal states to `API.md` and `ARCHITECTURE.md`
- freeze sell semantics in docs and tests until fractional sells are intentionally designed
- map user-visible failure behavior to backend error codes for launch-critical flows

Exit criteria:

- no trust-sensitive flow depends on contradictory docs or inferred frontend behavior

### Workstream 3 - UX State Hardening

DRI:

- Frontend lead

Delivery target:

- before final manual QA pass

Goal:

- make the shipped frontend explain failure and in-between states as intentionally as success states

Required work:

- verify disabled, empty, partial, stale, and retry states on auth, markets, trade, sell, portfolio, create, resolve, and dispute flows using the `QA.md` checklist
- confirm portfolio language stays calm and trustworthy even with sell support
- remove any leftover debug-only behavior or ambiguous UI feedback before release cut

Exit criteria:

- a first-time internal user can always tell what happened, what failed, and what to do next

### Workstream 4 - Verification And Release Gating

DRI:

- Eng lead + QA owner

Delivery target:

- before release candidate signoff

Goal:

- make launch confidence come from proof, not optimism

Required work:

- map each release gate to an owner and verification artifact in this plan or a linked checklist
- keep `go test ./...` as the minimum automated backend shipping baseline
- define the frontend shipping baseline explicitly: production build plus manual QA checklist for launch-critical flows
- add any missing manual verification steps required by shipped sell behavior and dispute review behavior

Exit criteria:

- every launch gate has a human owner and a concrete proof artifact

### Workstream 5 - Deployment And Rollback Readiness

DRI:

- Eng lead + ops owner

Delivery target:

- before launch rehearsal

Goal:

- ensure launch is reversible and production behavior is knowable

Required work:

- document production deployment steps for backend and frontend in `OPERATIONS.md` or linked release docs
- define version bump expectations tied to `VERSION` and `CHANGELOG.md`
- document rollback expectations for code, routing, and data-sensitive failures
- add a launch rehearsal checklist with participants, pass/fail criteria, and outputs

Exit criteria:

- the team can rehearse deploy + rollback without inventing steps live

---

## Sequencing

```text
1. Lock launch control and trust contracts
      |
      v
2. Resolve routing / API boundary ambiguity
      |
      v
3. Harden UX states and launch-critical operator visibility
      |
      v
4. Complete verification artifacts and release gates
      |
      v
5. Run launch rehearsal, then ship to first internal users
```

Recommended order:

1. route / API namespace decision as prerequisite input to contract freeze
2. `Launch Control` and `Trust Contracts`
3. Day-1 operations coverage
4. UX state audit for launch-critical flows
5. release rehearsal and final checklist

### Launch Rehearsal Checklist

Pass only if the team can complete all of the following without improvisation:

- deploy the release candidate through the intended production-like path
- verify auth, trade, sell, resolve, dispute, admin dispute review, and WebSocket flows
- inspect the expected logs, metrics, and alerts for at least one success path and one forced failure path
- execute the rollback procedure and confirm the system returns to the prior known-good state

Required participants:

- engineering owner
- frontend or product owner
- whoever will own day-1 operations or incident response

---

## Artifact Owners

| Artifact | Owner | Needed before |
|---|---|---|
| `API.md` launch contract updates | backend lead | API contract gate signoff |
| `OPERATIONS.md` ops matrix and deployment notes | eng lead + ops owner | launch rehearsal |
| `QA.md` launch checklist and frontend manual baseline | QA owner + frontend lead | release candidate signoff |
| trust state machines in docs | product + backend lead | product trust gate |
| portfolio valuation / PnL presentation rules | frontend lead + product owner | UX state gate |
| rehearsal record | eng lead | final ship decision |

---

## Weekly Execution Table

| Workstream | Immediate blocker | Owner | Due before |
|---|---|---|---|
| Routing And Contract Cleanup | route compatibility decision | Eng lead | API contract freeze |
| Trust-Sensitive Rule Lock | final dispute lifecycle wording | Product + backend lead | trust gate signoff |
| UX State Hardening | launch-critical state audit complete | Frontend lead | final manual QA pass |
| Verification And Release Gating | release artifacts mapped to owners | Eng lead + QA owner | release candidate signoff |
| Deployment And Rollback Readiness | deployment + rollback steps documented | Eng lead + ops owner | launch rehearsal |

### Weekly Operating Cadence

- Monday: update workstream status, blockers, and artifact ownership
- Wednesday: resolve cross-doc contradictions and contract drift
- Friday: review gate status and decide whether the branch is closer to rehearsal or blocked

---

## Risks That Matter Most

- the team keeps treating the system as “almost there” and launches without explicit trust contracts
- routing ambiguity creates subtle production bugs or deployment confusion
- sell support exists in code and UI but remains under-specified in edge cases
- dispute and payout handling remain understandable only to current maintainers
- deployment succeeds but post-launch debugging is too weak to respond confidently

---

## Nice-To-Have After Launch Readiness

- fractional-share sell support
- calibration scores
- seasonal tournaments
- market archival
- expanded admin and audit tooling

---

## Source Documents This Plan Depends On

- `README.md`
- `API.md`
- `ARCHITECTURE.md`
- `DESIGN.md`
- `USER_FLOWS.md`
- `QA.md`
- `OPERATIONS.md`
- `TODOS.md`
- `CHANGELOG.md`

If any of those documents disagree with this plan, the disagreement should be resolved before launch rather than tolerated.

---

## Locked Decisions From CEO Review

- auth and market-creation contracts are launch-critical and explicitly part of the release-gate model
- final ship authority is shared by the engineering lead and product owner; disagreement defaults to no-ship
- portfolio valuation and PnL presentation are frozen as part of the UX state gate and artifact ownership model
- progress is tracked through a weekly operating cadence rather than vague relative timing alone
