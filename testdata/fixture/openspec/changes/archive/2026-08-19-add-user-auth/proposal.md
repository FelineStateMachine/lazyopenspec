## Why

Tally currently trusts a shared API token per team, so we cannot tell members apart, audit who changed what, or restrict exports to owners. Individual accounts are a prerequisite for assignment, notifications, and per-user permissions on the Q3 roadmap.

## What Changes

- Add email + password accounts with argon2id password hashing
- Add cookie-based sessions with an inactivity timeout and an optional "remember me" extension
- Lock accounts temporarily after repeated failed logins
- **BREAKING**: Retire the shared team API token; existing integrations must switch to personal access tokens (tracked separately)

## Capabilities

### New Capabilities
- `user-auth`: Account login, session lifecycle, and brute-force protection

### Modified Capabilities
<!-- None: task-management already assumes an authenticated member -->

## Impact

- New `users` and `sessions` tables (migrations 0031-0033)
- `api/middleware/auth.go` replaced; every handler now receives a `*User`
- Web UI gains login/logout pages
- Shared team token removed from `config/tally.toml`
