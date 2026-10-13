# Change: Full-text task search

## Why

Projects with more than a few hundred tasks are hard to navigate: the only way to find a task is to page through the list. Support tickets about "lost" tasks have doubled since September, and two enterprise trials cited missing search as a blocker.

## What Changes

- Add a full-text search endpoint over task titles and descriptions, scoped to projects the user belongs to
- Rank results by relevance with a recency tie-breaker
- Task listing accepts a `status` filter so search results and lists share the same filtering vocabulary

## Capabilities

### New Capabilities
- `search`: Full-text search over tasks the user can access

### Modified Capabilities
- `task-management`: Task listing gains a `status` filter

## Impact

- Postgres `tsvector` column + GIN index on `tasks` (migration 0041)
- New `GET /api/search?q=` endpoint
- Search box in the web UI header
