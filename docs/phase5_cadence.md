# Phase 5 Item 7 - Polling Cadence & Pings

## Cadence
- Base UISP poll interval configurable; faster on changes, slower on errors.
- Per-role ping intervals: backbone/AP every 60s; configurable overrides per site.

## Error/Diagnostics
- If cadence constraints violated, log and surface banner “polling limited” with reason (rate limit, backoff).
