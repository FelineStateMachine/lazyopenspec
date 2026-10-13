## Context

Sessions are opaque ids stored in Postgres (`sessions` table, see add-user-auth). Expiry is enforced only by `last_used_at + timeout`. There is no absolute cap and no user-facing session management. The pentest report (PT-2026-09, finding H-2) recommends idle <= 12h and absolute <= 7d.

## Goals / Non-Goals

**Goals:**
- Meet the pentest recommendation for idle and absolute session lifetime
- Give users self-service revocation of other sessions
- Ship without forcing everyone to log in at deploy time

**Non-Goals:**
- Device fingerprinting or anomaly detection
- Changing the session id format or cookie attributes
- Admin-initiated revocation of other users' sessions (follow-up)

## Decisions

- **Store `absolute_expires_at` instead of deriving from `created_at`**: lets us backfill existing sessions to `now() + 7d` so nobody is logged out on deploy, and leaves room for per-org policies later. Alternative: compute from `created_at` - rejected because it would instantly expire most existing sessions.
- **Revocation deletes the row**: simpler than a `revoked_at` tombstone; audit log already records the event.
- **Drop `remember_me` in two steps**: stop reading it now, drop the column next release, so a rollback does not need a down-migration.

## Risks / Trade-offs

- [More frequent logins annoy users] → Ship with in-app notice; measure login volume for two weeks.
- [Backfill touches every session row] → Run in batches of 5,000 during low traffic.
- [Clock skew between app servers] → All comparisons use database `now()`.

## Migration Plan

1. Deploy migration 0042 (add `absolute_expires_at`, nullable) and backfill.
2. Deploy middleware enforcing both timeouts; treat NULL as `created_at + 7d`.
3. Next release: make the column NOT NULL and drop `remember_me`.

Rollback: revert middleware deploy; the extra column is harmless.

## Open Questions

- Should revoking the current session from the sessions page log the user out immediately, or require confirmation?
