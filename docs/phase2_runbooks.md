# Phase 2 Item 9 - Runbooks for Device Actions

## Actions Covered
- Simulate outage.
- Acknowledge/clear incidents.
- Enter/exit maintenance windows.
- Send test notification.

## Runbook Outline
- Preconditions (auth role, environment, feature flags).
- Steps to perform action via UI/API/CLI.
- Expected results and state changes.
- Rollback/cleanup steps.
- Troubleshooting with verbose error references (codes, request IDs).

## Error/Banner Expectations
- Each action surfaces success/failure banner with request_id.
- On failure, show actionable message (missing role, device not found, conflicting maintenance) and link to logs/diagnostics.
