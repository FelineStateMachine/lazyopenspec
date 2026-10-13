## 1. Session storage

- [x] 1.1 Add nullable `absolute_expires_at` to `sessions` (migration 0042)
- [x] 1.2 Backfill `absolute_expires_at = now() + interval '7 days'` in batches of 5,000

## 2. Expiry and revocation

- [x] 2.1 Enforce 12h idle timeout in session middleware
- [ ] 2.2 Enforce 7-day absolute lifetime (treat NULL as `created_at + 7d`)
  - [x] Unit test: session used 1h ago but created 8 days ago is rejected
  - [ ] Unit test: NULL `absolute_expires_at` falls back to `created_at`
- [ ] 2.3 Add `GET /account/sessions` and `DELETE /account/sessions/:id`
- [ ] 2.4 Build Active sessions page with "This device" label

## 3. Remove remember-me

- [X] 3.1 Remove "Remember me" checkbox from login form
- [ ] 3.2 Stop reading `remember_me` in session creation

## 4. Rename and docs

- [ ] 4.1 Rename `password_login` metrics and log fields to `password_auth`
- [ ] 4.2 Update security docs and support macro for shorter sessions
