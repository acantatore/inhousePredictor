# Contributing

Thanks for contributing to InhousePredictor.

This project aims to feel safe to change: code, tests, and docs should move together.

## Prerequisites

You need:

- Go 1.24+
- Node.js 20+
- npm 10+
- Docker + Docker Compose

## Local Setup

1. Clone the repo and create a local env file:

```bash
git clone <your-repo-url>
cd inhousePredictor
cp .env.example .env
```

2. Set at least `JWT_SECRET` in `.env`.

3. Start Postgres and the API:

```bash
docker compose up --build -d
```

4. In a second terminal, start the frontend:

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

5. Open the app at `http://localhost:5173`.

## Running Tests

Backend tests:

```bash
go test ./...
```

See `TESTING.md` for the testing philosophy and conventions used in this repo.

## Development Expectations

- keep business rules enforced in the backend, not only in the UI
- when fixing a bug, add a regression test if the bug is testable
- when adding a new function or conditional branch, add or update tests accordingly
- never commit code that makes existing tests fail
- keep docs in sync when behavior, setup, or contracts change

## Docs To Update When Behavior Changes

Depending on the change, check:

- `README.md` for feature, setup, and doc index updates
- `API.md` for request and response contract changes
- `ARCHITECTURE.md` and `DATA_MODEL.md` for backend and schema drift
- `QA.md` and `TESTING.md` for verification changes
- `CHANGELOG.md` and `VERSION` for release-facing updates

## Pull Request Checklist

Before opening or updating a PR, make sure you:

- ran `go test ./...`
- built the frontend if you changed frontend code:

```bash
cd frontend
npm run build
```

- updated any stale docs touched by your change
- kept user-facing copy plain and product-oriented

## Release Notes

This repo now tracks release labels in `VERSION` and user-facing changes in `CHANGELOG.md`.
If your change affects shipped behavior, add or update the relevant changelog entry.
