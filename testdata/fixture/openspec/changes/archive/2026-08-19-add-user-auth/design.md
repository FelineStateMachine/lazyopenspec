## Context

All requests are authenticated today by a single bearer token stored in `config/tally.toml`. There is no `users` table. The web UI is a server-rendered Go app, so cookie sessions are simpler than JWTs.

## Goals / Non-Goals

**Goals:**
- Per-user identity on every request
- Sessions that survive browser restarts when the user opts in
- Basic brute-force protection without external infrastructure

**Non-Goals:**
- SSO / OAuth providers (follow-up)
- Personal access tokens for the API (separate change)
- Multi-factor authentication

## Decisions

- **argon2id over bcrypt**: argon2id is memory-hard and is the OWASP 2024 recommendation; `golang.org/x/crypto/argon2` is already vendored.
- **Opaque server-side sessions over JWT**: we need instant revocation and already have Postgres; a 32-byte random session id in an `HttpOnly; Secure; SameSite=Lax` cookie keeps the client dumb.
- **Lockout counter in Postgres, not Redis**: login volume is tiny; avoiding a new dependency wins.

## Risks / Trade-offs

- [Lockout can be abused to deny service to a known email] → Lockout is time-boxed to 15 minutes and does not reveal whether the account exists.
- [Session table grows unbounded] → Nightly job deletes sessions idle past expiry.
