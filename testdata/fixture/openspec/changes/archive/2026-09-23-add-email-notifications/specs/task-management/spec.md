## MODIFIED Requirements

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
