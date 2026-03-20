# InhousePredictor Copy Guide

> Last updated: 2026-03-20

---

## Purpose

This guide defines the product voice and UI copy rules for InhousePredictor.

The product should feel like an internal company tool for better judgment, not a trading terminal or gambling app.

---

## Voice

The voice is:

- warm
- clear
- team-oriented
- trustworthy
- plainspoken

The voice is not:

- hype-driven
- promotional
- casino-like
- crypto-native
- overloaded with trading jargon

---

## Core Tone Rules

- explain actions in plain language
- prefer clarity over cleverness
- make first-time participation feel safe
- sound like a useful internal tool, not a finance product
- keep confidence calm, not flashy

---

## Preferred Terms

Use:

- `Buy YES`
- `Buy NO`
- `Points to spend`
- `Estimated shares`
- `Awaiting resolution`
- `Closing soon`
- `Recently resolved`
- `Resolver`
- `Evidence`
- `Dispute`

---

## Terms To Avoid

Avoid:

- AMM
- reserve ratio
- invariant
- long
- short
- position sizing
- alpha
- edge
- ape in
- moon
- jackpot
- bet big

If technical terms are necessary in developer-facing docs, keep them out of the normal user interface.

---

## Product Framing

Preferred framing:

- shared company forecasting
- internal prediction market
- better judgment through visible forecasts
- see what the crowd currently thinks
- understand what needs attention now

Avoid framing the product as:

- a money-making system
- a competition-first leaderboard tool
- a crypto exchange clone in the interface language

---

## Action Copy Rules

- CTAs should be direct and readable
- timing context should appear close to the primary action
- when actions are disabled, explain why
- success states should confirm what changed

Examples:

- `Buy YES`
- `Buy NO`
- `Create market`
- `Resolve market`
- `Submit dispute`

---

## Governance And Trust Copy

These concepts should always be explained clearly:

- resolver responsibility
- evidence requirement
- dispute availability and next steps
- creator trading restriction
- market close and resolve times

Preferred phrasing for creator restriction:

- `Creators cannot trade their own markets.`

Preferred phrasing for unavailable trading:

- `Trading is closed for this market.`
- `You cannot trade a market you created.`
- `You do not have enough points for this trade.`
- `This market is busy right now. Please try again.`

---

## Empty State Rules

Every empty state should include:

- context
- warmth
- one clear next action

Good pattern:

- `No markets are open right now. You can create one when your team has a question worth forecasting.`

Bad pattern:

- `No items found.`

---

## Error Copy Rules

- be specific about what happened
- avoid exposing technical internals
- focus on what the user can do next when possible

Good examples:

- `That email is already registered.`
- `Please enter a valid evidence link.`
- `This dispute window has closed.`
- `You need more points to make this trade.`

Bad examples:

- `40001 serialization error`
- `pq: duplicate key value violates unique constraint`

---

## Success Copy Rules

- confirm the action cleanly
- avoid celebratory finance language
- use calm feedback

Examples:

- `Trade placed.`
- `Market created.`
- `Resolution recorded.`
- `Dispute submitted.`

---

## Market Writing Rules

- market questions should be concrete and resolvable
- descriptions should explain context, not add jargon
- deadlines should be explicit
- resolution language should point users toward evidence and next steps

---

## Accessibility And Clarity Notes

- do not rely on color alone to express YES, NO, warning, or status
- status labels should include words, not just visual treatment
- live updates should be informative without feeling noisy

---

## Copy Review Checklist

- does this sound like an internal company tool?
- would a first-time employee understand it?
- does it avoid trader or crypto slang?
- does it explain disabled or error states clearly?
- does it reinforce trust where outcomes and disputes matter?
