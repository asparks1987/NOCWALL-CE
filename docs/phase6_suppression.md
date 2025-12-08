# Phase 6 Item 3 - Dependency-Based Suppression

## Behavior
- If upstream gateway offline, suppress downstream AP/switch incidents/alerts.
- Record suppression with parent device id and reason.

## Error/Diagnostics
- UI shows badge “suppressed (upstream offline)” with parent info and request_id.
- Logs include suppression decisions for audits.
