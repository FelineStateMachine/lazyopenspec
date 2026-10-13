# notifications Specification

## Purpose
TBD - created by archiving change add-email-notifications. Update Purpose after archive.
## Requirements
### Requirement: Assignment notification email
The system SHALL send an email to a task's new assignee within 60 seconds of the task being assigned to them.

#### Scenario: Task assigned to another member
- **WHEN** Alice assigns a task to Bob
- **THEN** Bob receives an email within 60 seconds containing the task title and a link to the task

#### Scenario: Self-assignment
- **WHEN** a member assigns a task to themselves
- **THEN** no notification email is sent

### Requirement: Notification preferences
Users SHALL be able to opt out of notification emails for each project they belong to, and the system MUST honor the opt-out for all subsequent notifications.

#### Scenario: Opted-out user is assigned a task
- **WHEN** Bob has opted out of emails for project "Website" and is assigned a task in "Website"
- **THEN** no email is sent to Bob

#### Scenario: Opt-out is per project
- **WHEN** Bob has opted out for project "Website" but not for project "Mobile"
- **THEN** Bob still receives assignment emails for tasks in "Mobile"

