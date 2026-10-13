## 1. Infrastructure

- [x] 1.1 Add `outbox` and `notification_preferences` tables (migration 0038)
- [x] 1.2 Add SMTP settings to config and deployment secrets

## 2. Notifier

- [x] 2.1 Emit assignment events to the outbox inside the assignment transaction
- [x] 2.2 Implement `cmd/notifier` worker that drains the outbox and sends email
- [x] 2.3 Skip self-assignments and opted-out recipients

## 3. Preferences UI

- [x] 3.1 Add per-project "Email me about assignments" toggle to project settings
- [x] 3.2 Add integration test covering opt-out
