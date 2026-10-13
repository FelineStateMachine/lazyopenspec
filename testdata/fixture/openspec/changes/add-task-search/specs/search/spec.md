## ADDED Requirements

### Requirement: Full-text task search
The system SHALL return tasks whose title or description match a free-text query, limited to projects the requesting user is a member of.

#### Scenario: Match in title
- **WHEN** a member searches for "invoice"
- **THEN** results include every accessible task whose title contains "invoice" or a stemmed variant such as "invoices"

#### Scenario: Inaccessible project
- **WHEN** a task matching the query belongs to a project the user is not a member of
- **THEN** that task is not included in the results

#### Scenario: Empty query
- **WHEN** a member submits a query that is empty after trimming whitespace
- **THEN** the system responds with a 422 validation error

### Requirement: Search result ranking
Search results MUST be ordered by relevance score, with ties broken by most recently updated first, and SHALL return at most 25 results per page.

#### Scenario: Title match outranks description match
- **WHEN** task A matches the query in its title and task B matches only in its description
- **THEN** task A appears before task B

#### Scenario: Tie broken by recency
- **WHEN** two tasks have the same relevance score
- **THEN** the more recently updated task appears first
