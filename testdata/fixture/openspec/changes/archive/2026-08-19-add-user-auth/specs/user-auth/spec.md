## ADDED Requirements

### Requirement: Password login
The system SHALL authenticate users with an email address and password, storing only argon2id password hashes.

#### Scenario: Valid credentials
- **WHEN** a user submits a registered email and the correct password
- **THEN** the system creates a session and sets an `HttpOnly`, `Secure` session cookie
- **AND** redirects the user to their last visited project

#### Scenario: Invalid credentials
- **WHEN** a user submits an unknown email or a wrong password
- **THEN** the system responds with the generic message "Invalid email or password"
- **AND** does not reveal which field was wrong

### Requirement: Session expiry
Sessions SHALL expire after 30 days without activity, after which the user MUST log in again.

#### Scenario: Expired session
- **WHEN** a request arrives with a session cookie last used more than 30 days ago
- **THEN** the system deletes the session and redirects to the login page

#### Scenario: Activity extends session
- **WHEN** a request arrives with a session last used 29 days ago
- **THEN** the request succeeds and the session's last-used time is updated

### Requirement: Remember me
The login form SHALL offer a "Remember me" option that extends the session inactivity timeout to 90 days.

#### Scenario: Remember me selected
- **WHEN** a user logs in with "Remember me" checked
- **THEN** the session inactivity timeout is 90 days instead of 30

### Requirement: Account lockout
The system MUST lock an account for 15 minutes after 5 consecutive failed login attempts.

#### Scenario: Fifth failed attempt
- **WHEN** a user fails to log in 5 times in a row
- **THEN** further login attempts for that account are rejected for 15 minutes
- **AND** the response is indistinguishable from an invalid-credentials response

#### Scenario: Successful login resets counter
- **WHEN** a user fails to log in 3 times and then logs in successfully
- **THEN** the failed-attempt counter resets to zero
