# InhousePredictor API Contract

> Last updated: 2026-03-27

---

## Purpose

This document describes the target API contract for InhousePredictor v1.
It is based on the current backend in `ARCHITECTURE.md`, with launch-required adjustments called out where the implementation still needs to catch up.

---

## API Principles

- authenticated employees are the default audience
- responses should be JSON and consistent across handlers
- trust-sensitive failures should be explicit and safe
- product copy should avoid exposing internal trading jargon unnecessarily
- current implementation gaps are not considered final contract decisions

---

## Authentication

### Scheme

- Bearer token using JWT HS256
- token expiry: 24 hours
- token contains `user_id` and `is_admin`

### Header

```http
Authorization: Bearer <jwt>
```

### Notes

- registration and login are public
- all product endpoints other than auth require authentication
- WebSocket requires authentication and approved origins

---

## Conventions

### Content Type

- request and response bodies use `application/json`

### Timestamps

- all timestamps should be ISO 8601 / RFC 3339 strings

### IDs

- all resource IDs are UUID v4 strings

### Money-Like Values

- user balances, trade cost, and initial liquidity are integer point values
- buy-side trade `cost` is integer points to spend
- sell-side trade `cost` is the whole-number share amount to sell under the current API contract
- points are internal only and not redeemable for cash

---

## Target Response Shape

The current codebase standardizes HTTP responses around a JSON success and error envelope.

### Success Envelope

```json
{
  "data": {}
}
```

For list responses:

```json
{
  "data": []
}
```

### Error Envelope

```json
{
  "error": {
    "code": "market_closed",
    "message": "This market is closed for trading."
  }
}
```

This envelope is now the implementation target for all handlers.

---

## Public Endpoints

### `POST /auth/register`

Create a new user.

#### Request

```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "correct horse battery staple"
}
```

#### Expected Behavior

- creates a user with a hashed password
- initializes balance to `10000` points
- duplicate email should return a conflict error

#### Target Errors

- `email_already_registered`
- `validation_error`

### `POST /auth/login`

Authenticate a user.

#### Request

```json
{
  "email": "jane@example.com",
  "password": "correct horse battery staple"
}
```

#### Response Data

```json
{
  "token": "<jwt>",
  "user": {
    "id": "<uuid>",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "balance": 10000,
    "is_admin": false
  }
}
```

#### Target Errors

- `invalid_credentials`

---

## Authenticated Endpoints

### `GET /me`

Return the current authenticated user.

#### Response Data

```json
{
  "id": "<uuid>",
  "name": "Jane Doe",
  "email": "jane@example.com",
  "balance": 8700,
  "is_admin": false
}
```

### `GET /markets`

List markets.

#### Query Params

- `category=<people|okrs|slas|financials|general>`
- `status=<open|closed|resolved|disputed|cancelled>`

#### Notes

- current implementation bounds responses to 50 rows
- cursor pagination should still be added before broader rollout

### `POST /markets`

Create a market.

#### Request

```json
{
  "question": "Will the support backlog stay under 200 tickets by quarter end?",
  "description": "Context for why this market matters.",
  "category": "slas",
  "resolver_id": "<uuid>",
  "initial_liquidity": 500,
  "closes_at": "2026-04-01T12:00:00Z",
  "resolves_at": "2026-04-03T12:00:00Z"
}
```

#### Expected Behavior

- debits creator balance by initial liquidity
- creates market and one-to-one pool
- market begins at 50/50 pricing
- creator is not allowed to trade this market after creation

#### Target Errors

- `validation_error`
- `insufficient_balance`
- `unauthorized`

### `GET /markets/{id}`

Return a single market, including current state needed for market detail.

#### Should Include

- question and description
- category and status
- creator and resolver identity
- close and resolve times
- current prices
- evidence and outcome if resolved
- dispute state
- aggregate activity as needed for UI

#### Target Errors

- `not_found`

### `POST /markets/{id}/resolve`

Resolve a market.

#### Authorization

- resolver only

#### Request

```json
{
  "outcome": "yes",
  "evidence_url": "https://example.com/evidence"
}
```

#### Expected Behavior

- records outcome
- records evidence URL
- sets resolution timestamps and dispute deadline
- leaves payout pending until the dispute deadline passes without dispute, or until an admin completes dispute review

#### Target Errors

- `unauthorized`
- `invalid_market_state`
- `validation_error`

### `POST /markets/{id}/dispute`

Submit a dispute.

#### Request

```json
{
  "reason": "The outcome appears incorrect because the linked evidence does not support it."
}
```

#### Expected Behavior

- accepts disputes only while eligible
- records dispute reason, user, and timestamps

#### Target Errors

- `dispute_window_closed`
- `invalid_market_state`
- `validation_error`

### `POST /markets/{id}/review-dispute`

Review a disputed market.

#### Authorization

- admin only

#### Request

```json
{
  "action": "override_outcome",
  "outcome": "no",
  "evidence_url": "https://example.com/updated-evidence"
}
```

#### Supported Actions

- `confirm_original`
- `override_outcome`
- `cancel_market`

#### Expected Behavior

- only works for disputed markets
- records dispute review actor and timestamp
- if confirming or overriding, finalizes payout idempotently
- if cancelling, refunds spent points idempotently

#### Target Errors

- `forbidden`
- `invalid_market_state`
- `validation_error`

### `POST /markets/{id}/trade`

Buy or sell YES/NO shares with points.

#### Request

```json
{
  "side": "yes",
  "cost": 1000
}
```

**Request semantics:**
- buy orders use `cost` as integer points to spend
- sell orders use `cost` as the whole-number share amount to sell

**Side values:**
- `"yes"` — Buy YES shares
- `"no"` — Buy NO shares  
- `"sell_yes"` — Sell YES shares (must have position)
- `"sell_no"` — Sell NO shares (must have position)

#### Expected Behavior

- checks market is open and before `closes_at`
- checks user has sufficient balance (for buys) or sufficient shares (for sells)
- rejects creator self-trading
- executes trade atomically
- updates position and pool state
- emits WS price update

#### Response Should Include

- trade id
- shares received
- cost spent
- price before and after
- updated balance

#### Target Errors

- `market_closed`
- `insufficient_balance`
- `creator_cannot_trade_own_market`
- `market_busy`
- `validation_error`

### `GET /markets/{id}/trades`

Return recent market trades.

#### Notes

- current implementation returns the last 50 trades
- UI should treat this as supporting data, not a named public trade tape

### `GET /positions`

Return the authenticated user's holdings across markets.

#### Should Include

- market references
- yes shares and no shares
- market status
- resolution state where relevant
- enough context to render portfolio views

---

## WebSocket Protocol

### Endpoint

```http
GET /ws
GET /ws?market_id=<uuid>
```

### Subscription Modes

- no `market_id`: subscribe to all market updates
- with `market_id`: subscribe to one market

### Launch-Ready Requirements

- require authentication
- enforce approved company origins
- support both global and market-scoped subscriptions

### Server Message Shape

```json
{
  "type": "price_update",
  "market_id": "<uuid>",
  "payload": {
    "yes_price": 0.65,
    "no_price": 0.35,
    "last_trade": {
      "side": "yes",
      "shares": 10.5,
      "cost": 1000
    }
  }
}
```

### Notes

- slow clients may be dropped
- there is no persistence or replay model in v1
- live updates should complement, not replace, normal page loads

---

## Domain Objects

### User

```json
{
  "id": "<uuid>",
  "name": "Jane Doe",
  "email": "jane@example.com",
  "balance": 10000,
  "is_admin": false,
  "created_at": "2026-03-20T00:00:00Z"
}
```

### Market

```json
{
  "id": "<uuid>",
  "question": "Will the support backlog stay under 200 tickets by quarter end?",
  "description": "Context for why this market matters.",
  "category": "slas",
  "status": "open",
  "outcome": null,
  "creator_id": "<uuid>",
  "resolver_id": "<uuid>",
  "initial_liquidity": 500,
  "closes_at": "2026-04-01T12:00:00Z",
  "resolves_at": "2026-04-03T12:00:00Z",
  "resolved_at": null,
  "dispute_deadline": null,
  "evidence_url": null,
  "yes_price": 0.5,
  "no_price": 0.5
}
```

### Trade

```json
{
  "id": "<uuid>",
  "market_id": "<uuid>",
  "user_id": "<uuid>",
  "side": "yes",
  "shares": 10.5,
  "cost": 1000,
  "yes_price_before": 0.5,
  "yes_price_after": 0.65,
  "created_at": "2026-03-20T00:00:00Z"
}
```

### Position

```json
{
  "market_id": "<uuid>",
  "yes_shares": 10.5,
  "no_shares": 0,
  "market_status": "open"
}
```

### Dispute

```json
{
  "id": "<uuid>",
  "market_id": "<uuid>",
  "user_id": "<uuid>",
  "reason": "The evidence does not support the stated outcome.",
  "created_at": "2026-03-20T00:00:00Z",
  "resolved_at": null,
  "resolved_by": null
}
```

---

## Permission Summary

- authenticated employee: browse, trade, create market, view own positions, dispute eligible markets
- resolver: all employee rights plus resolve assigned markets
- admin: all employee rights plus dispute queue review and dispute handling

---

## Known Gaps Between Current Implementation And Target Contract

- market listing is bounded but does not yet expose cursor pagination
- payout finalization is lazy after dispute deadline or admin review rather than background-job driven
- no frontend is consuming this contract yet, so end-to-end UI/API compatibility is still pending
