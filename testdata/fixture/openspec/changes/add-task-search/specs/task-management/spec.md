## MODIFIED Requirements

### Requirement: Task listing
The system SHALL list a project's tasks ordered by most recently updated first, paginated at 50 tasks per page, and SHALL accept an optional `status` filter that limits results to tasks in the given status.

#### Scenario: First page of tasks
- **WHEN** a member requests the task list for a project with 120 tasks
- **THEN** the system returns the 50 most recently updated tasks
- **AND** includes a `next_cursor` for the following page

#### Scenario: Empty project
- **WHEN** a member requests the task list for a project with no tasks
- **THEN** the system returns an empty list and no `next_cursor`

#### Scenario: Filter by status
- **WHEN** a member requests the task list with `status=in_progress`
- **THEN** only tasks whose status is `in_progress` are returned

#### Scenario: Invalid status filter
- **WHEN** a member requests the task list with `status=blocked`
- **THEN** the system responds with a 422 validation error
