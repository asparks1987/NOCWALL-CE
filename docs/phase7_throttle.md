# Phase 7 Item 3 - Dedupe & Throttle

## Behavior
- Collapse duplicate alerts within window; channel-specific throttles.
- Respect suppression and maintenance; do not send muted alerts.

## Error/Diagnostics
- Debug view shows dedupe decisions and throttle reasons; include request_id for troubleshooting.
