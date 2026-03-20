# InhousePredictor Security Model

> Last updated: 2026-03-20

---

## Purpose

This document defines the security posture, trust boundaries, and hardening requirements for InhousePredictor.

The system is an internal company tool, but it still handles authentication, authorization, balances, and trust-sensitive market outcomes. Internal-only does not remove the need for strong server-side controls.

---

## Security Principles

- enforce trust rules on the server
- keep private user state private
- avoid leaking internal implementation details to end users
- minimize operational secrets exposure
- harden real-time channels to the same standard as HTTP APIs

---

## Auth Model

- authentication uses JWT HS256
- tokens expire after 24 hours
- `JWT_SECRET` is provided by environment variable
- tokens include `user_id` and `is_admin`

### Implications

- stolen tokens remain valid until expiry unless additional revocation logic is added later
- admin role changes require a new token because role state is embedded at issue time

---

## Authorization Model

### Authenticated Employee

Can:

- browse markets
- trade eligible markets
- create markets
- view own positions and portfolio
- file disputes while eligible

Cannot:

- resolve markets they are not assigned to
- perform admin dispute actions
- trade markets they created

### Resolver

Can:

- perform all normal employee actions
- resolve assigned markets with evidence

### Admin

Can:

- perform all normal employee actions
- review and act on disputes through the focused dispute workflow

---

## Trust Boundaries

### Public To Authenticated Users

- market question
- category
- status
- YES and NO prices
- deadlines
- creator identity where relevant
- resolver identity
- evidence and outcome
- dispute state
- aggregated activity

### Private To The User

- own balance impact
- own positions
- own trade confirmations
- own portfolio details

### Restricted To Admins

- dispute workflow actions and queue details beyond normal user context

---

## Critical Product Security Rules

- creators cannot trade their own markets
- trades must not execute after `closes_at`
- disputes must not be accepted after `dispute_deadline`
- only assigned resolvers can resolve a market
- evidence is required for resolution
- payout must be safe to retry and must not double-credit balances

These are product rules and security rules at the same time because violating them breaks trust and fairness.

---

## Secrets Handling

- `JWT_SECRET` must be set outside source control
- `.env.example` can document required keys, but real values must not be committed
- future operational secrets should follow the same pattern

---

## Password Handling

- passwords are hashed with bcrypt
- plaintext passwords must never be logged or stored
- minimum password length validation should be added before launch

---

## WebSocket Security

### Current Risk

The current WS endpoint is public and accepts any origin.

### Required Launch State

- require the same auth boundary as the HTTP API
- restrict origins to approved company domains
- test both market-scoped and all-market subscriptions

### Reason

Real-time price and activity data is part of the internal product surface and should not be exposed through a weaker channel than the main API.

---

## Error And Information Exposure

### Current Risk

- duplicate email can leak raw Postgres errors
- serialization failures can leak raw DB errors
- handlers use inconsistent error paths

### Required Launch State

- consistent JSON errors
- user-safe messages for validation, auth, conflicts, busy state, and not-found responses
- no raw database or stack details in normal client responses

---

## Abuse And Misuse Cases

### Creator Self-Dealing

Risk:

- creators can exploit framing or insider knowledge if allowed to trade own markets

Mitigation:

- explicitly forbidden policy
- server-side trade rejection
- UI disclosure at creation and trade surfaces

### Trading After Deadline

Risk:

- users can act on information that should no longer affect the market

Mitigation:

- enforce `closes_at` server-side
- test the edge around the deadline

### Incorrect Or Unsupported Resolution

Risk:

- users lose trust if outcomes are not backed by evidence or are impossible to challenge

Mitigation:

- required evidence URL
- visible resolver identity
- dispute flow
- admin dispute review workflow

### Concurrency And Replay Issues

Risk:

- concurrent trade attempts can lead to race conditions or confusing failures

Mitigation:

- serializable transactions
- row locking
- retry policy with safe busy response

### Excessive Or Automated Requests

Risk:

- no rate limiting currently exists

Mitigation:

- add rate limiting or equivalent request protection before broader rollout if abuse becomes likely

---

## Security Gaps Before Launch

- unauthenticated and unrestricted WS access
- raw DB error leakage in user-facing flows
- no server-side dispute deadline enforcement yet
- no server-side trade close enforcement yet
- no password minimum validation yet
- no rate limiting

---

## Validation And Input Hardening

- enforce max length for question and description
- validate evidence URLs
- enforce password minimum length
- validate timing relationships such as `closes_at < resolves_at`

---

## Audit And Observability Needs

- structured logs for trade execution
- structured logs for payout processing
- structured logs for dispute actions once implemented
- no secrets or plaintext passwords in logs

---

## Minimum Security Release Gate

Before first users:

- WS auth and origin restriction are enforced
- creator self-trading is blocked server-side
- close and dispute deadlines are enforced server-side
- payout is idempotent
- raw DB errors are no longer exposed
- auth and role checks are covered by tests
