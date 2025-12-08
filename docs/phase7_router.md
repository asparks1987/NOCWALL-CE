# Phase 7 Item 1 - Notification Router

## Behavior
- Consumes incident events; fans out to channels (Gotify, FCM, email, SMS/webhook).
- Templates per channel; actionable where supported.
- Retries with backoff; dead-letter on repeated failures.

## Error/Diagnostics
- Logs include channel, incident_id, request_id, error details; UI shows delivery status and last error banner.
