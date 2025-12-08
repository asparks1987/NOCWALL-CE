# Phase 6 Item 4 - Escalation Schedules & Repeats

## Behavior
- Repeat alerts per policy (e.g., every 10m) until ack/resolution.
- Escalate to secondary/on-call lead after N repeats or time threshold.
- Respect quiet hours unless critical.

## Error/Diagnostics
- Logs and UI show next escalation time and history; banner includes last send and next planned send with request_id.
