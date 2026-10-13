## Why

A September penetration test flagged our 30-day idle sessions (90 days with "Remember me") as high risk: a stolen cookie stays valid for months and users have no way to sign out other devices. Our SOC 2 auditor wants a fix before the October audit window closes.

## What Changes

- Shorten the idle timeout to 12 hours and add a 7-day absolute session lifetime
- Let users see and revoke their active sessions
- **BREAKING**: Remove the "Remember me" option; all sessions follow the same lifetime rules
- Rename the "Password login" requirement to "Password authentication" to match the security glossary

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `user-auth`: Session expiry rules tightened, session revocation added, remember-me removed, password login renamed

## Impact

- `sessions` table gains `absolute_expires_at`; `remember_me` column dropped one release later
- Session middleware enforces both timeouts
- New "Active sessions" page under account settings
- Users will be signed out more often; support macro needed
