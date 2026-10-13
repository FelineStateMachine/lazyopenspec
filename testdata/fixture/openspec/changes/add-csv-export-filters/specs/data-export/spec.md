## ADDED Requirements

### Requirement: Export filters
The CSV export SHALL accept optional `status`, `assignee`, and `updated_since` filters, and MUST include only tasks matching every supplied filter.

#### Scenario: Filter by status and assignee
- **WHEN** an owner exports with `status=done&assignee=bob`
- **THEN** the CSV contains only tasks that are `done` and assigned to Bob

#### Scenario: Filtered file name
- **WHEN** an owner exports project "website" with `status=done`
- **THEN** the downloaded file is named `website-tasks-status-done.csv`

#### Scenario: Invalid date filter
- **WHEN** an owner exports with `updated_since=yesterday`
- **THEN** the system responds with a 422 validation error naming the `updated_since` parameter

## MODIFIED Requirements

### Requirement: CSV export
The system SHALL export a project's tasks as an RFC 4180 CSV file with a header row. By default the columns are `id`, `title`, `status`, `assignee`, `created_at`, and `updated_at`; when a `columns` parameter is supplied the system SHALL emit exactly those columns in the given order.

#### Scenario: Export a project
- **WHEN** a project owner requests a CSV export of a project with 3 tasks
- **THEN** the system returns a CSV file with a header row and 3 data rows
- **AND** the response has content type `text/csv; charset=utf-8`

#### Scenario: Escape commas and quotes
- **WHEN** a task title contains a comma or a double quote
- **THEN** the field is wrapped in double quotes and embedded quotes are doubled

#### Scenario: Custom column selection
- **WHEN** an owner exports with `columns=title,status`
- **THEN** the CSV header row is exactly `title,status` and each row has two fields

#### Scenario: Unknown column
- **WHEN** an owner exports with `columns=title,priority`
- **THEN** the system responds with a 422 validation error naming `priority`
