# Task Management Specification

## Purpose

Defines how Tally members create, update, assign, and list tasks within a project. Tasks are the core unit of work in Tally; every other capability (notifications, export, search) builds on the task model described here.
## Requirements
### Requirement: Task creation
The system SHALL allow an authenticated project member to create a task with a title of 1-200 characters and an optional Markdown description of up to 10,000 characters.

#### Scenario: Create task with title only
- **WHEN** a member submits a new task with title "Write release notes" and no description
- **THEN** the system creates the task with status `todo`
- **AND** returns the task with a generated id and `created_at` timestamp

#### Scenario: Reject empty title
- **WHEN** a member submits a new task with an empty or whitespace-only title
- **THEN** the system rejects the request with a 422 validation error
- **AND** no task is created

### Requirement: Task status transitions
Tasks SHALL move through the statuses `todo`, `in_progress`, and `done`, and the system MUST reject any transition to a status outside that set.

#### Scenario: Complete a task
- **WHEN** a member sets the status of an `in_progress` task to `done`
- **THEN** the task status becomes `done`
- **AND** `completed_at` is set to the current time

#### Scenario: Reopen a completed task
- **WHEN** a member sets the status of a `done` task to `todo`
- **THEN** the task status becomes `todo`
- **AND** `completed_at` is cleared

#### Scenario: Unknown status
- **WHEN** a member sets a task status to `blocked`
- **THEN** the system rejects the request with a 422 validation error

### Requirement: Task assignment
A task SHALL have at most one assignee, and the assignee MUST be a member of the task's project. Changing the assignee SHALL emit an assignment event consumed by the notifications capability.

#### Scenario: Assign to project member
- **WHEN** a member assigns a task to another member of the same project
- **THEN** the task's assignee is updated
- **AND** an assignment event is recorded for the new assignee

#### Scenario: Assign to non-member
- **WHEN** a member assigns a task to a user who is not in the project
- **THEN** the system rejects the request with a 403 error
- **AND** the existing assignee is unchanged

#### Scenario: Unassign a task
- **WHEN** a member clears the assignee of a task
- **THEN** the task has no assignee
- **AND** no assignment event is recorded

### Requirement: Task listing
The system SHALL list a project's tasks ordered by most recently updated first, paginated at 50 tasks per page.

#### Scenario: First page of tasks
- **WHEN** a member requests the task list for a project with 120 tasks
- **THEN** the system returns the 50 most recently updated tasks
- **AND** includes a `next_cursor` for the following page

#### Scenario: Empty project
- **WHEN** a member requests the task list for a project with no tasks
- **THEN** the system returns an empty list and no `next_cursor`

