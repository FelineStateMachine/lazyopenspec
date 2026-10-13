# Data Export Specification

## Purpose

Lets project owners take their data out of Tally in open formats, either for reporting in spreadsheets or for migrating to another tool. Exports always reflect the project's tasks at the moment the export is requested.

## Requirements

### Requirement: CSV export
The system SHALL export all of a project's tasks as an RFC 4180 CSV file with a header row containing `id`, `title`, `status`, `assignee`, `created_at`, and `updated_at`.

#### Scenario: Export a project
- **WHEN** a project owner requests a CSV export of a project with 3 tasks
- **THEN** the system returns a CSV file with a header row and 3 data rows
- **AND** the response has content type `text/csv; charset=utf-8`

#### Scenario: Escape commas and quotes
- **WHEN** a task title contains a comma or a double quote
- **THEN** the field is wrapped in double quotes and embedded quotes are doubled

### Requirement: JSON export
The system SHALL export a project's tasks as a JSON array where each element contains every task field, including the full description.

#### Scenario: Export as JSON
- **WHEN** a project owner requests a JSON export
- **THEN** the system returns `application/json` containing an array of task objects

### Requirement: Export authorization
Only project owners MUST be able to request an export of a project's data.

#### Scenario: Non-owner attempts export
- **WHEN** a project member who is not an owner requests an export
- **THEN** the system rejects the request with a 403 error
- **AND** no export file is generated
