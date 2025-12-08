# Phase 4 Item 6 - Mobile Bootstrap Endpoint

## Endpoint
- `GET /mobile/config` providing UISP URL, API base, feature flags, push registration URL, environment, versions, banner messages.
- Requires auth; returns request_id for diagnostics.

## Behavior
- Caches config briefly; forces refresh on version change.
- Provides endpoints for test notification and connectivity checks.

## Error/Banner Expectations
- On failure, respond with specific code (`mobile_config_missing`, `mobile_config_unavailable`) and message; UI/Android banner shows request_id and retry guidance.
