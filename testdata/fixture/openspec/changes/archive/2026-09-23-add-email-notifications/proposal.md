# Change: Email notifications for task assignment

## Why

Members only discover newly assigned tasks when they happen to open Tally, so assignments sit unnoticed for days. Now that every request has a real user (add-user-auth), we can tell people when work lands on their plate.

## What Changes

- Send an email to the assignee whenever a task is assigned or reassigned to them
- Let users opt out of notification emails per project
- Assignment now has a side effect, so the task-management assignment requirement gains a notification scenario

## Capabilities

### New Capabilities
- `notifications`: Outbound email notifications and per-project preferences

### Modified Capabilities
- `task-management`: Task assignment now triggers an assignee notification

## Impact

- New `notification_preferences` table
- New background worker `cmd/notifier` reading from the `outbox` table
- SMTP credentials added to deployment secrets
