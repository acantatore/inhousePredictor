# Changelog

All notable changes to this project will be documented in this file.

The format is loosely based on Keep a Changelog and uses semantic versioning for release labels.

## [Unreleased]

### Docs
- added release-management files: `VERSION`, `CHANGELOG.md`, and `CONTRIBUTING.md`
- expanded `README.md` to make release and contribution docs easier to discover

## [0.1.0] - 2026-03-27

### Added
- email/password authentication with JWT-based sessions and authenticated user profile access
- market browsing, market detail, create market, resolve market, dispute submission, and admin dispute review flows
- YES / NO trading with buy support and sell support for whole-share sell orders
- portfolio and position tracking for open and resolved market participation
- authenticated, origin-restricted WebSocket updates for market activity and price changes
- backend unit and integration coverage for core market, trade, auth, and payout behavior

### Changed
- standardized the project docs around the shipped frontend foundation and current runtime behavior
- documented the current sell contract as whole-share only until the trade API supports fractional sell amounts

### Known Limitations
- sell orders currently use whole-share amounts only
- market list pagination remains bounded rather than cursor-based
- production frontend deployment guidance is still incomplete
