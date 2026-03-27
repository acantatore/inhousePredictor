# InhousePredictor Decisions

> Last updated: 2026-03-27

---

## Purpose

This file records active product, policy, and technical decisions for InhousePredictor.
It is the current source of truth for choices that shape implementation.

When a decision changes, update this file instead of relying on informal discussion.

---

## Decision Format

Each decision includes:

- status: `active`, `deferred`, or `open`
- date: when the decision was locked or recorded
- context: why this matters
- decision: the choice being made
- consequences: what the team must build, enforce, or accept

---

## Active Decisions

### D-001 - Internal play-money product only

- Status: `active`
- Date: 2026-03-20
- Context: The product is intended for internal company forecasting, not regulated financial trading.
- Decision: InhousePredictor uses internal points only. It does not support real-money deposits, withdrawals, or cash-equivalent rewards.
- Consequences:
  - User balances are measured in internal points.
  - Product copy should avoid gambling and trading language.
  - Security, compliance, and operations should be designed for an internal tool, not a financial platform.

### D-002 - Company-authenticated employees are the primary user base

- Status: `active`
- Date: 2026-03-20
- Context: The design system and architecture define the product as a private company workspace.
- Decision: The product is built for authenticated employees inside a company context, with additional resolver and admin roles.
- Consequences:
  - There is no anonymous browsing model in v1.
  - Product flows should assume a known company audience.
  - Access control should be role-aware without becoming an enterprise permission matrix in v1.

### D-003 - Any authenticated employee can create markets

- Status: `active`
- Date: 2026-03-20
- Context: Market creation is a primary participation flow, not an admin-only action.
- Decision: Any authenticated employee can create a market.
- Consequences:
  - Market creation must be simple and well-guided.
  - The backend must enforce creation validation and liquidity debiting rules.
  - The UI must explain resolver responsibility and timeline fields clearly.

### D-004 - Creators cannot trade their own markets

- Status: `active`
- Date: 2026-03-20
- Context: Trust is a core product requirement. Allowing creators to trade their own markets creates framing and information asymmetry concerns.
- Decision: Market creators are not allowed to trade on markets they created.
- Consequences:
  - The backend must enforce this rule server-side.
  - The market detail view must explain the restriction when relevant.
  - The create market flow must disclose this before submission.
  - QA and security coverage must include regression tests for creator self-trading attempts.

### D-005 - Resolver identity is always visible

- Status: `active`
- Date: 2026-03-20
- Context: Resolution trust depends on users understanding who is responsible for calling the outcome.
- Decision: Each market must display the assigned resolver identity.
- Consequences:
  - Market detail and resolution-related flows should surface resolver identity prominently.
  - API responses must expose resolver identity where needed for UI rendering.

### D-006 - Evidence is required for market resolution

- Status: `active`
- Date: 2026-03-20
- Context: Resolved outcomes must be explainable and auditable by normal employees.
- Decision: Resolver-submitted outcomes require an `evidence_url`.
- Consequences:
  - The resolve endpoint and UI must require evidence.
  - Resolved market views should make evidence easy to find.
  - QA should treat missing or invalid evidence as a release-blocking defect.

### D-007 - Normal users do not see a named public trade tape

- Status: `active`
- Date: 2026-03-20
- Context: The design system calls for aggregated activity, not a public feed of who traded what.
- Decision: Normal authenticated users can see market prices, timing, status, and aggregated activity, but not a named public trade tape.
- Consequences:
  - UI should present market activity in aggregate form.
  - Private trade confirmations and portfolio data remain user-scoped.
  - Admin-only audit surfaces can be added later if needed.

### D-008 - Admin v1 is a focused dispute queue, not a full admin suite

- Status: `active`
- Date: 2026-03-20
- Context: The product needs a dispute workflow before broad internal use, but not a broad management console.
- Decision: The only planned admin surface in v1 is a dispute-review workflow and queue.
- Consequences:
  - User management and broad moderation tooling are out of scope for now.
  - Admin API and UI work should focus on dispute handling only.

### D-009 - Buy-first copy with sell support in v1

- Status: `active`
- Date: 2026-03-20
- Context: The product now supports both buying and selling positions, but the interface still needs to feel like a calm internal forecasting tool rather than a trading terminal.
- Decision: v1 trading keeps spend-first buy language while also supporting sell flows using shares-to-sell language.
- Consequences:
  - UI copy should use `Points to spend` and `Estimated shares` for buys.
  - Sell flows should use `Shares to sell` and clear proceeds guidance.
  - Portfolio UX must still avoid trader-style live PnL framing.

### D-010 - Use CPMM for pricing

- Status: `active`
- Date: 2026-03-20
- Context: The existing trading system is built around a constant-product market maker.
- Decision: Markets use a constant-product market maker with equal initial YES/NO reserves derived from the initial liquidity.
- Consequences:
  - Prices are implied from reserves, not manually set.
  - CPMM behavior requires unit tests and careful concurrency handling.
  - Product copy should hide AMM terminology from normal users.

### D-011 - Trade execution uses serializable transactions plus row locking

- Status: `active`
- Date: 2026-03-20
- Context: Concurrent trading must not corrupt balances or pool state.
- Decision: Trades execute at PostgreSQL `SERIALIZABLE` isolation and lock the pool row for update.
- Consequences:
  - Concurrency conflicts are expected and must be handled cleanly.
  - The API should retry serialization failures, then return a user-safe busy response.
  - Integration tests must cover concurrent trade behavior.

### D-012 - Payouts must become idempotent before launch

- Status: `active`
- Date: 2026-03-20
- Context: The architecture doc identifies non-idempotent payout behavior as a must-fix launch blocker.
- Decision: Market payout processing must be idempotent and safe to retry.
- Consequences:
  - Schema and service changes are required before first users.
  - QA must include a regression test that repeated payout attempts do not double-credit balances.

### D-013 - Shared JSON error responses are part of the target API contract

- Status: `active`
- Date: 2026-03-20
- Context: The current handlers leak raw DB and HTTP errors in several paths.
- Decision: The target API contract uses consistent JSON success and error responses across handlers.
- Consequences:
  - Duplicate email, insufficient balance, validation failure, busy market, auth failure, and not-found states should all return standardized responses.
  - API docs should describe the target contract, even where the current implementation still needs cleanup.

### D-014 - WebSocket access must be authenticated and origin-restricted before launch

- Status: `active`
- Date: 2026-03-20
- Context: The current WS endpoint is too permissive for an internal company product.
- Decision: The launch-ready WebSocket model requires the same authentication boundary as the HTTP API and a company origin allowlist.
- Consequences:
  - Current public WS behavior is considered pre-launch only.
  - Security, API, QA, and operations docs should treat authenticated WS as required launch work.

### D-015 - Migrations must run as part of normal application startup

- Status: `active`
- Date: 2026-03-20
- Context: Current migration behavior depends on fresh container startup and is not operationally reliable enough.
- Decision: The application should run migrations programmatically on startup before serving traffic.
- Consequences:
  - Deployment no longer depends on special container initialization behavior.
  - Operations docs should describe startup migrations as the target operating model.

---

## Deferred Decisions

### D-016 - Fractional-share sell mechanics

- Status: `deferred`
- Date: 2026-03-20
- Context: Sell functionality now exists, but the current API contract still reuses integer `cost` for sell orders, which limits sells to whole-share amounts.
- Decision: Fractional-share sell support is deferred until the API contract changes.
- Consequences:
  - Current UI and docs should describe sell inputs as whole-share amounts.
  - Later work must revisit handler, service, and contract semantics before fractional sells ship.

### D-017 - Full admin suite

- Status: `deferred`
- Date: 2026-03-20
- Context: The system may eventually need broader moderation and user management, but that is not part of v1.
- Decision: Full administrative tooling is deferred.
- Consequences:
  - No general-purpose user management console is planned right now.
  - Admin docs should stay focused on disputes and operational exceptions.

---

## Open Decisions

### D-018 - Final dispute terminal states and admin actions

- Status: `open`
- Date: 2026-03-20
- Context: The roadmap identifies admin dispute resolution as required, but the exact lifecycle is not fully locked.
- Decision: Still open.
- Consequences:
  - API, data model, and user-flow docs should document the expected workflow and mark final state transitions as pending implementation details.
  - Before launch, the team must define whether admins can confirm the original outcome, replace the outcome, cancel the market, or support multiple resolution variants.

### D-019 - Error response envelope shape

- Status: `open`
- Date: 2026-03-20
- Context: The project needs standardized JSON errors, but the final envelope shape is not yet implemented.
- Decision: Still open.
- Consequences:
  - Docs should recommend a simple stable envelope and note that code must align.
  - Tests and frontend work should avoid depending on raw `http.Error` behavior.

### D-020 - Pagination model for market list

- Status: `open`
- Date: 2026-03-20
- Context: The current market list returns all rows, and the roadmap calls for keyset pagination.
- Decision: The exact cursor contract is not yet finalized.
- Consequences:
  - API and plan docs should treat pagination as required before broader rollout.
  - Frontend list design should avoid assuming unlimited result sets.

---

## Change Log

- 2026-03-20: Created initial decision log from `ARCHITECTURE.md`, `DESIGN.md`, and `TODOS.md`.
- 2026-03-20: Locked creator trading policy as `not allowed`.
