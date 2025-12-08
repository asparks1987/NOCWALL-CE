# Phase 2 Item 6 - Incident Timeline & Audit Log Schema

## Incident Timeline
- Timeline entries: incident_id, type (triggered/ack/clear/escalate/suppress/resolve/notification), actor (user/system), timestamp, details (thresholds, channel, suppression reason), correlation_id.
- Exposed via `GET /incidents/{id}/timeline` for UI/Android; supports pagination.
- Include request_id for each action to tie to logs.

## Audit Log
- Records all privileged actions (login, token create/revoke, policy changes, maintenance edits, device overrides, notification settings changes).
- Fields: id, user_id, action, target_type/id, payload snapshot, ip/user-agent (when available), created_at, request_id.
- Stored immutably; retention per compliance.

## Error/Diagnostics
- Failed writes/logging emit structured errors with request_id; banner surfaces logging failures to admins.
- Provide “export diagnostics” for an incident (timeline + related alerts) with correlation IDs.
