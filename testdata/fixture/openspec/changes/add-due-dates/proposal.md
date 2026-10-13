## Why

Teams use Tally for release checklists, but tasks have no notion of a deadline, so people track dates in task titles ("[Fri] ship notes"). That breaks sorting and makes overdue work invisible.

## What Changes

- Tasks gain an optional `due_date` (calendar date, no time component, interpreted in the project's time zone)
- Task listing can sort by due date and show an "overdue" badge
- Assignees get a reminder notification the morning a task is due

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `task-management`: Task creation accepts `due_date`; task listing supports `sort=due_date`
- `notifications`: New due-date reminder notification

## Impact

- `tasks.due_date DATE NULL` column and index
- Project settings gain a time zone field (defaults to UTC)
- Notifier worker gains a daily scheduler
