## ADDED Requirements

### Requirement: Session revocation
Users SHALL be able to list their active sessions and revoke any of them, and a revoked session MUST be rejected on its next request.

#### Scenario: Revoke another device
- **WHEN** a user revokes a session belonging to another device from the Active sessions page
- **THEN** the next request using that session is redirected to the login page

#### Scenario: Session list contents
- **WHEN** a user opens the Active sessions page
- **THEN** each session shows its creation time, last-used time, and user agent
- **AND** the current session is labelled "This device"

## MODIFIED Requirements

### Requirement: Session expiry
Sessions SHALL expire after 12 hours without activity and MUST expire no later than 7 days after creation regardless of activity.

#### Scenario: Idle timeout
- **WHEN** a request arrives with a session last used more than 12 hours ago
- **THEN** the system deletes the session and redirects to the login page

#### Scenario: Activity extends idle window
- **WHEN** a request arrives with a session last used 11 hours ago
- **THEN** the request succeeds and the session's last-used time is updated

#### Scenario: Absolute lifetime reached
- **WHEN** a request arrives with a session created more than 7 days ago, even if it was used within the last hour
- **THEN** the system deletes the session and redirects to the login page

## REMOVED Requirements

### Requirement: Remember me
**Reason**: Long-lived sessions were flagged as high risk in pentest PT-2026-09 (finding H-2).
**Migration**: None for users; all sessions now follow the standard expiry rules. The `remember_me` column is dropped in the following release.

## RENAMED Requirements

- FROM: `### Requirement: Password login`
- TO: `### Requirement: Password authentication`
