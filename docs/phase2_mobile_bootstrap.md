# Phase 2 Item 8 - Mobile Config Endpoint

## Endpoint
- `GET /mobile/config`: returns UISP base URL, API base, feature flags, push registration URL, environment, version, and banner messages.
- Auth required; ties to user account; includes request_id.

## Push Registration
- `POST /push/register` with device token, platform (android/web), app version, locale, capabilities (actionable notifications support).
- Response includes correlation ID; errors are verbose with hints (invalid token, stale session).

## Error/Banner Expectations
- If config fetch fails, Android/web show banner with reason and retry guidance.
- Include “copy diagnostics” with request_id, http status, and payload snippet (sanitized).

## Security
- Tokens bound to user and environment; rotate on logout; revoke on device removal.
- Rate limit registration to prevent abuse; validate FCM tokens format.
