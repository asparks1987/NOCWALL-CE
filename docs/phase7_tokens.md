# Phase 7 Item 2 - Push Token Registration & Cleanup

## Behavior
- Register device/browser tokens with platform, env, user; validate format.
- Expire/remove stale tokens; rotate on logout; dedupe tokens per device.

## Error/Diagnostics
- Registration errors include request_id, reason (invalid token, expired), for banner display.
