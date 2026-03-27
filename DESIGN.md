# InhousePredictor Design System

> Last updated: 2026-03-27

---

## Product Thesis

InhousePredictor is a shared company forecasting space for normal employees, not a trader tool.
The interface should help people quickly understand what the company is predicting, what the crowd thinks, what needs attention now, and how to participate safely with play-money points.

The product should feel warm, trustworthy, and useful in day-to-day company life. It should not feel like a crypto terminal, a gambling app, or a generic SaaS dashboard.

---

## Design Principles

1. Clarity over cleverness.
2. Trust is visible at the screen level.
3. Probability is central, but jargon stays hidden.
4. Empty states are features, not placeholders.
5. Users should always know what they can do next.
6. Live updates should feel current, never noisy.
7. Warmth matters, but precision still matters.

---

## Primary Users

### Employee

- Browses markets
- Buys YES or NO with points
- Tracks open positions and resolved outcomes
- Files disputes when needed
- Can create markets directly

### Resolver

- Resolves assigned markets
- Provides evidence URL at resolution time
- Needs a lightweight, trustworthy resolution flow

### Admin

- Reviews disputed markets
- Handles operational edge cases
- Uses a focused dispute queue rather than a full admin suite in v1

---

## Product Tone

The product voice is warm, team-oriented, and clear.

- It should sound like an internal company tool for better judgment.
- It should avoid trader slang, hype, or casino energy.
- It should explain actions plainly: `Buy YES`, `Buy NO`, `Points to spend`, `Awaiting resolution`.
- It should make participation feel safe for first-time users.

---

## Visual Direction

### Overall Feel

The UI should feel like a shared company forecasting workspace: calm, thoughtful, and friendly, while still data-literate.

Avoid:

- crypto aesthetics
- flashing tickers
- giant marketing heroes
- generic three-column card dashboards
- overly hard-edged finance styling

Prefer:

- structured boards and rows
- gentle sectioning
- soft but disciplined spacing
- subtle emphasis on timing and probability
- editorial warmth with operational clarity

### Color

- Base UI uses warm neutrals and soft slate tones.
- YES uses a clear green accent.
- NO uses a clear red accent.
- Disputed, warning, or closing-soon states use amber.
- System chrome, metadata, and dividers use slate/ink tones.

Color must support meaning, but meaning must never rely on color alone.

### Typography

- Use a humanist sans-serif for interface copy and long-form reading.
- Use a restrained monospace face for probabilities, balances, timestamps, and numeric summaries.
- Headings should feel calm and confident, not promotional.

### Motion

- Use small fades and soft emphasis for live updates.
- Use lightweight confirmation transitions after trades, creates, resolves, and disputes.
- Never use scrolling tickers, aggressive flashes, or decorative motion.

---

## Navigation Model

### Desktop

Top app bar:

- `Markets`
- `Portfolio`
- `Create Market`
- balance badge
- profile menu
- `Admin` only for admins

### Mobile

Bottom navigation:

- `Markets`
- `Portfolio`
- `Create`
- `Profile`

Do not use a left-side enterprise navigation rail in v1.

---

## Information Architecture

```text
App Shell
├── Markets
│   ├── Team context header
│   ├── Closing Soon
│   ├── Open
│   └── Recently Resolved
├── Market Detail
│   ├── Summary
│   ├── Timeline
│   ├── Evidence / outcome
│   ├── Aggregated activity
│   └── Trade panel
├── Portfolio
│   ├── Performance summary
│   ├── Awaiting resolution
│   ├── Open positions
│   └── Resolved outcomes
├── Create Market
├── Resolve Market
├── Dispute Submission
└── Admin Disputes
```

---

## Screen Blueprints

### 1. Markets

This is the default home screen.

#### Purpose

- Show what needs attention now
- Make the crowd signal legible
- Help users scan quickly
- Encourage informed participation

#### Structure

```text
Markets
├── Team context header
│   ├── short explanation of what prediction markets are
│   └── why they help the company make better decisions
├── Controls
│   ├── category filter
│   ├── status filter
│   └── Compact / Expanded view toggle
├── Closing Soon
├── Open
└── Recently Resolved
```

#### Ordering

Sections appear in this order:

1. `Closing Soon`
2. `Open`
3. `Recently Resolved`

Urgency should be visible before completeness.

#### Market Row

Compact row includes:

- market question
- category
- YES price
- NO price
- compact probability visualization
- close time
- status pill
- `your position` chip if relevant

Expanded row adds:

- short description preview
- stronger contextual metadata

Compact is the default. Expanded is an optional user-controlled mode.

### 2. Market Detail

This page should be explanation-first and action-ready.

#### Desktop Layout

Two-column layout:

- left: market understanding
- right: sticky trade ticket

#### Mobile Layout

- one-column content flow
- persistent bottom action area for trading

#### Content Order

1. Market question
2. Category and status
3. Close time and resolve time
4. Resolver identity
5. Description and context
6. Timeline
7. Evidence and outcome, when resolved
8. Dispute state or dispute action
9. Aggregated activity

#### Trade Panel

The trade panel should remain visible on desktop as the user scrolls.

It includes:

- buy / sell mode toggle when the user has a position
- YES / NO selection
- points-to-spend or shares-to-sell input
- estimated shares preview for buys and estimated proceeds preview for sells
- current balance
- close-time reminder
- clear success/error feedback

When trading is unavailable, the panel becomes an explanation surface instead of a dead form.

Disabled reasons include:

- market closed
- market resolved
- creator cannot trade own market
- unauthorized user
- insufficient balance
- temporary market-busy state

### 3. Portfolio

Portfolio is performance-first.

#### Purpose

- help users understand how they are doing
- reinforce learning and participation
- show what is still unresolved

#### Structure

Top summary cards or stat blocks show:

- resolved winnings or losses
- hit rate / forecast accuracy
- points currently committed
- markets awaiting resolution

Supporting sections:

- `Awaiting Resolution`
- `Open Positions`
- `Resolved Outcomes`

Do not imply trader-style liquid mark-to-market PnL just because the product now supports selling.

### 4. Create Market

Create Market is a primary employee flow, not an admin surface.

#### Pattern

Single-page guided form with strong sectioning.

#### Sections

- Question
- Context
- Resolver
- Timing
- Liquidity
- Review

#### Guidance

The form should explain:

- what makes a good market question
- what the resolver is responsible for
- what close time means
- what resolve time means
- that initial liquidity comes from the creator's balance
- that creators cannot trade their own markets

### 5. Resolve Market

Resolver-only flow.

Requirements:

- clear confirmation of outcome
- required evidence URL
- plain explanation of the dispute window after resolution

### 6. Dispute Submission

Dispute is a trust-preserving action, not an edge-case afterthought.

Requirements:

- only visible while eligible
- explain what happens after a dispute is filed
- focused reason input
- clear success confirmation

### 7. Admin Disputes

Planned as a focused queue, not a full admin control center.

Each row should show:

- market question
- dispute reason
- creator and resolver
- timestamps
- evidence state
- status
- action CTA

---

## Component Vocabulary

- `Market Row`
- `Probability Split`
- `Status Pill`
- `Trade Ticket`
- `Deadline Block`
- `Evidence Card`
- `Dispute Banner`
- `Activity Summary`
- `Position Chip`
- `Balance Badge`
- `Performance Stat`

New UI should reuse this vocabulary instead of inventing ad hoc patterns.

---

## Interaction Rules

- Use plain-language trading copy.
- Buy flows stay spend-first; sell flows use shares-to-sell language.
- Show estimated shares before confirmation.
- Keep timing context near the primary CTA.
- Replace disabled action states with explanations.
- Use calm, in-place live updates.
- Keep all market actions understandable for a first-time employee.

### Trade Language

Preferred:

- `Buy YES`
- `Buy NO`
- `Sell YES`
- `Sell NO`
- `Points to spend`
- `Shares to sell`
- `Estimated shares`
- `Awaiting resolution`

Avoid:

- AMM terminology
- reserve terminology
- invariant language
- trader shorthand

---

## Public, Private, and Restricted Information

### Public To Authenticated Users

- market question
- category
- status
- YES / NO prices
- deadlines
- creator identity where relevant
- resolver identity
- evidence and outcome
- dispute state
- aggregated activity

### Private To The User

- their positions
- their trade confirmations
- their balance impact
- their portfolio details

### Restricted / Admin

- deeper dispute workflow details
- later audit-oriented activity if needed

Normal users should not see a named public trade tape.

---

## Trust And Governance Rules

- Any authenticated employee can create markets.
- Creators cannot trade their own markets.
- This restriction should be explained clearly in the market detail view and market creation review block.
- Resolver identity should always be visible.
- Evidence should be prominent after resolution.
- Deadlines should never be hidden in low-priority metadata.
- Dispute status should explain what is happening and what happens next.

---

## Interaction State Matrix

| Surface | Loading | Empty | Error | Success | Partial / Live |
|---|---|---|---|---|---|
| Markets | Skeleton rows | Explain what markets are, CTA to create | Failed to load markets | Filter/update applied | Prices stale or live feed disconnected |
| Market Detail | Skeleton header and trade panel | N/A | Market missing, closed, unauthorized, busy | Trade placed, dispute filed, resolved | Core content loaded but activity feed unavailable |
| Portfolio | Skeleton stat blocks and rows | No positions yet, CTA to explore markets | Failed to load portfolio | N/A | Some metrics loaded, some details delayed |
| Create Market | Disabled submit while validating | N/A | Inline field validation or submit failure | Market created | Draft-like local state retained after recoverable error |
| Resolve Market | Disabled submit | N/A | Invalid evidence, unauthorized, bad state | Resolution recorded | Confirmation shown while downstream refresh catches up |
| Admin Disputes | Skeleton queue | No disputes to review | Failed to load disputes | Dispute action completed | Queue data stale or refreshing |

### Empty State Rules

Every empty state must include:

- warmth
- context
- one clear next action

Never ship `No items found.` as the full design.

---

## Responsive Behavior

### Desktop

- Markets use dense rows for quick scanning.
- Market detail uses a two-column layout with sticky trade panel.
- Portfolio summary appears above detailed lists.

### Tablet

- Compress spacing carefully.
- Keep key market metadata visible.
- Allow detail view to switch between compressed two-column and stacked layouts depending on width.

### Mobile

- Single-column flow by default.
- Market detail uses a persistent bottom action area.
- Filters collapse into a sheet.
- Expanded rows remain readable without becoming long cards.

Responsive design is not just stacking. Each viewport should feel intentionally designed.

---

## Accessibility Requirements

- Minimum touch target size: 44px
- Full keyboard access for filters, forms, tabs, and actions
- Strong visible focus states
- Status colors must always include text labels
- Live updates must use accessible announcements without overwhelming screen readers
- Form validation must be announced clearly and placed near relevant fields
- Contrast must remain readable in all state treatments

---

## Screen-Level Hierarchy Rules

### Markets

Users should notice, in order:

1. What needs attention now
2. What the crowd currently thinks
3. Whether they already have a stake

### Market Detail

Users should notice, in order:

1. What this market is asking
2. Whether it is still actionable
3. What the current crowd signal is
4. Why they should trust the eventual outcome
5. What action they can take right now

### Portfolio

Users should notice, in order:

1. How they are doing overall
2. Which predictions are still unresolved
3. What resolved recently

---

## What Already Exists

The current backend already defines the core product surfaces:

- Auth: register, login, me
- Markets: list, create, get, resolve, dispute
- Trading: trade execution, market trades, positions
- WebSocket updates: all markets or single market subscription

This design system should align with the current product architecture rather than inventing a different product.

---

## V1 Scope

### In Scope

- Login and registration
- Markets home
- Market detail
- Trading flow
- Portfolio
- Create market flow
- Resolve market flow
- Dispute submission
- Basic admin dispute queue planning and implementation

### Out Of Scope

- Public marketing site
- Full user management console
- Advanced charting
- Tournaments
- Calibration leaderboards
- Expert-mode compact trading UI
- Social feed mechanics

---

## Not In Scope For This Version Of The Design System

- Final visual token values
- exact type ramp values
- exact spacing scale values
- illustration system
- brand assets or logo work
- advanced analytics modules

These can be added later without changing the core product experience defined here.
