# InhousePredictor User Flows

> Last updated: 2026-03-27

---

## Purpose

This document defines the core user journeys for InhousePredictor v1.
It combines the intended experience from `DESIGN.md` with the current backend capabilities and launch-ready policy rules.

---

## Roles

- employee
- resolver
- admin

All roles are authenticated users. Resolver and admin capabilities are additional permissions, not separate product silos.

---

## Flow 1 - Register

### User

Employee

### Goal

Create an account and enter the product.

### Happy Path

1. User enters name, email, and password.
2. System validates the input.
3. System creates account with `10000` starting points.
4. User is ready to log in.

### Failure States

- email already registered
- invalid input
- backend unavailable

### UX Requirements

- explain the product in simple internal-tool language
- make duplicate email and password rules clear

---

## Flow 2 - Log In

### User

Employee

### Goal

Authenticate and reach the markets experience.

### Happy Path

1. User enters email and password.
2. System returns bearer token and user profile.
3. User lands in the authenticated app shell.

### Failure States

- invalid credentials
- expired or invalid token on future requests

### UX Requirements

- keep login form simple
- use clear, non-technical auth failure copy

---

## Flow 3 - Browse Markets

### User

Employee

### Goal

Understand what the company is predicting and what needs attention now.

### Happy Path

1. User lands on `Markets`.
2. System shows sections in this order:
   - `Closing Soon`
   - `Open`
   - `Recently Resolved`
3. User filters by category or status if needed.
4. User scans question, category, prices, close time, and status.
5. User opens a market detail page.

### Failure States

- no markets exist yet
- filters return no matching results
- market list fails to load
- live price updates disconnect or go stale

### UX Requirements

- make urgency visible before completeness
- show probability clearly without trading jargon
- empty states should explain what markets are and offer a clear next action

---

## Flow 4 - Open Market Detail

### User

Employee

### Goal

Understand one market and decide whether action is available.

### Happy Path

1. User opens a market detail page.
2. System shows question, category, status, deadlines, resolver, description, timeline, and current signal.
3. User sees sticky trade panel on desktop or persistent action area on mobile.
4. User either trades, reviews evidence, or checks dispute state depending on market status.

### Failure States

- market not found
- unauthorized access
- supporting activity feed unavailable while core content still loads

### UX Requirements

- explanation first, action second
- make resolver identity and timing easy to notice
- when trade is unavailable, replace dead controls with reasons

---

## Flow 5 - Buy YES Or NO

### User

Employee

### Goal

Express a forecast by spending internal points.

### Preconditions

- user is authenticated
- market is open and before `closes_at`
- user is not the market creator
- user has sufficient balance

### Happy Path

1. User chooses `Buy YES` or `Buy NO`.
2. User enters `Points to spend`.
3. System previews estimated shares.
4. User confirms trade.
5. System executes the trade atomically.
6. User sees confirmation with cost, shares, and updated balance.
7. Market prices update in place.

### Failure States

- market closed
- market resolved
- creator trying to trade own market
- insufficient balance
- market temporarily busy due to serialization conflicts
- live refresh lag after success

### UX Requirements

- use spend-first language
- keep close-time reminder near CTA
- always explain disabled or rejected states in plain language

---

## Flow 6 - View Portfolio

### User

Employee

### Goal

Understand positions, committed points, and resolved outcomes.

### Happy Path

1. User opens `Portfolio`.
2. System shows summary blocks for performance, committed points, and unresolved markets.
3. User reviews sections:
   - `Awaiting Resolution`
   - `Open Positions`
   - `Resolved Outcomes`

### Failure States

- no positions yet
- portfolio fails to load
- partial data loads

### UX Requirements

- avoid brokerage-style live PnL framing even though users can now sell positions
- emphasize learning and resolution status over trader-style analytics

---

## Flow 7 - Create Market

### User

Employee

### Goal

Create a new forecasting market without admin involvement.

### Happy Path

1. User opens `Create Market`.
2. User fills sections for question, context, resolver, timing, and liquidity.
3. System explains what makes a good market and what the resolver is responsible for.
4. Review step reminds the user that initial liquidity is deducted from their balance.
5. Review step also states that creators cannot trade their own markets.
6. User submits.
7. System debits liquidity, creates market, and returns the new market.

### Failure States

- insufficient balance
- invalid timing
- invalid resolver selection
- validation or submit failure

### UX Requirements

- this should feel like a primary product flow, not an admin tool
- validation should be inline and specific

---

## Flow 8 - Resolve Market

### User

Resolver

### Goal

Resolve an assigned market with evidence.

### Preconditions

- user is the assigned resolver
- market is in a resolvable state

### Happy Path

1. Resolver opens resolve flow.
2. Resolver chooses outcome.
3. Resolver enters required evidence URL.
4. System confirms dispute window expectations.
5. Resolver submits.
6. System records outcome, evidence, timestamps, and payout safely.

### Failure States

- unauthorized resolver attempt
- invalid evidence URL
- invalid market state
- payout-related backend failure

### UX Requirements

- make this flow feel serious and trustworthy
- do not hide the evidence requirement

---

## Flow 9 - Submit Dispute

### User

Employee

### Goal

Challenge an incorrect or unsupported market resolution.

### Preconditions

- user is authenticated
- market is in an eligible dispute state
- dispute deadline has not passed

### Happy Path

1. User sees dispute action while eligible.
2. User opens dispute form.
3. User enters focused reason.
4. System confirms what happens next.
5. Dispute is recorded and visible to admins.

### Failure States

- dispute window closed
- market not eligible for dispute
- validation failure

### UX Requirements

- dispute should feel trust-preserving, not hidden or hostile
- success confirmation should explain next steps

---

## Flow 10 - Review Disputes

### User

Admin

### Goal

Review disputed markets and complete the governance workflow.

### Happy Path

1. Admin opens dispute queue.
2. System shows market question, dispute reason, creator, resolver, evidence state, timestamps, and current status.
3. Admin opens a dispute record.
4. Admin applies the supported dispute action.
5. System records actor and timestamps.

### Failure States

- unauthorized access
- queue load failure
- stale dispute data
- unsupported or invalid transition

### UX Requirements

- queue should be focused, not cluttered with unrelated admin controls
- actions must be explicit and auditable

---

## State-Dependent Action Rules

### Trading Unavailable Reasons

The UI should explain why trading is unavailable when any of these apply:

- market closed
- market resolved
- creator cannot trade own market
- user is not authenticated
- insufficient balance
- market temporarily busy

### Resolution Visibility Rules

- resolver identity should always be visible
- evidence should be prominent after resolution
- dispute status should explain what is happening and what happens next

---

## Empty, Error, And Success State Rules

- empty states should include context and one clear next action
- validation should appear near the relevant field
- success states should confirm what changed
- partial or live-update issues should not block core content from rendering

---

## Mobile And Desktop Notes

- desktop market detail should use a two-column layout with sticky trade panel
- mobile should use a single-column flow with persistent bottom action area
- responsive behavior should preserve meaning, not just stack components
