# Security TODOs

Findings from security audit on 2026-03-25.

---

## HIGH

- [ ] **SQL injection pattern** — `internal/trade/repository.go:479-480`
  Dynamic SQL column names built via `fmt.Sprintf()`. Currently safe (col is constrained), but fragile. Replace with explicit `CASE` statement to eliminate the pattern entirely.

- [ ] **Missing authorization on ProgramView** — `internal/forecast/handler.go:155-162`
  Any authenticated user can enumerate all programs' risk data. Add ownership/permission check before calling `h.svc.ProgramRiskView`.

- [ ] **No TLS** — `cmd/server/main.go:89`
  Server starts with plain HTTP. Configure TLS via reverse proxy or in-app for production.

---

## MEDIUM

- [ ] **JWT accepted via URL query param** — `internal/auth/jwt.go:58`
  `?token=...` leaks tokens into server logs, browser history, and proxies. Remove query param support; accept tokens via `Authorization` header and cookies only.

- [ ] **WebSocket token in URL** — `frontend/src/lib/api.ts:42-51`
  Token passed as `?token=` on WebSocket URL — same log/proxy exposure as above. Pass token via WebSocket subprotocol or first message instead.

- [ ] **JWT stored in localStorage** — `frontend/src/context/AuthContext.tsx:27,34`
  localStorage is accessible to any JS on the page (XSS risk). HttpOnly cookies are already set server-side — remove localStorage read/write and rely on cookies alone.

- [ ] **No rate limiting on auth endpoints**
  `/auth/login` and `/auth/register` have no rate limiting, enabling brute force. Add rate-limiting middleware on these routes.

- [ ] **Missing CSRF protection**
  SameSite=Lax helps but doesn't fully protect state-changing endpoints. Switch cookies to SameSite=Strict or implement CSRF tokens for POST/PUT/DELETE routes.

- [ ] **Long JWT lifetime** — `internal/auth/jwt.go:28`
  Tokens expire after 24 hours. Reduce to 15–60 minutes and implement refresh token rotation to limit exposure from stolen tokens.

---

## LOW

- [ ] **User list exposes emails** — `internal/user/handler.go:81-98`
  Any authenticated user can list all users including email addresses. Make endpoint admin-only or redact emails from the response.

- [ ] **Evidence URL not validated** — `internal/market/handler.go:115,146`
  `req.EvidenceURL` is passed through without URL format validation. Call `validate.URL()` on resolution endpoints.

- [ ] **Negative `limit` not rejected** — `internal/user/handler.go:82-90`
  `limit` is parsed from the query string but not checked for negative values before use. Add a `limit > 0` guard after parsing.

- [ ] **Hardcoded secrets in tests** — `internal/auth/jwt_test.go:15`, `internal/ws/handler_test.go:47`
  Test files use literals like `"secret"` and `"super-secret"`. Low risk (test-only), but replace with clearly fake constants or test fixtures.

- [ ] **`.env.example` shows real-looking credentials**
  The example value `postgres://app:secret@localhost:5432/inhousepredictor` uses real-looking credentials. Replace with obvious placeholders like `<password>`.
