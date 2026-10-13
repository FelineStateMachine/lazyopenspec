## 1. Data model

- [x] 1.1 Add `users` table with unique lower(email) index (migration 0031)
- [x] 1.2 Add `sessions` table keyed by random 32-byte id (migration 0032)
- [x] 1.3 Add `failed_logins` counter and `locked_until` to `users` (migration 0033)

## 2. Authentication

- [x] 2.1 Implement argon2id hashing helpers in `internal/auth/password.go`
- [x] 2.2 Implement login/logout handlers and session cookie
- [x] 2.3 Replace shared-token middleware with session middleware
- [x] 2.4 Implement lockout after 5 failed attempts

## 3. UI and cleanup

- [x] 3.1 Add login page with "Remember me" checkbox
- [x] 3.2 Add nightly job to purge expired sessions
- [x] 3.3 Remove shared team token from config and docs
