# Testing

100% test coverage is the key to great vibe coding. Tests let you move fast, trust your instincts, and ship with confidence - without them, vibe coding is just yolo coding. With tests, it's a superpower.

## Framework

- Go standard library `testing`
- `github.com/stretchr/testify` for readable assertions

## How To Run Tests

```bash
go test ./...
```

## Test Layers

- Unit tests: pure logic such as CPMM math and JWT handling; keep these close to the package they verify.
- Integration tests: repository and service behavior against Postgres; add these as the backend hardening work lands.
- Smoke tests: startup, migrations, bootstrap-admin, and WS handshake checks for launch paths.
- E2E tests: frontend user journeys once the frontend exists.

## Conventions

- Put tests next to the package they cover using `*_test.go` filenames.
- Prefer table-free direct tests when there are only a few meaningful cases.
- Use `require` for must-pass assertions that should stop the test immediately.
- When fixing a bug, add a regression test that reproduces the broken behavior.
- When adding a conditional branch, add tests for both paths.
