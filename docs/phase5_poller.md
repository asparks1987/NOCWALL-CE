# Phase 5 Item 1 - UISP Poller Service

## Behavior
- Poll UISP at configured cadence with retries/backoff.
- Normalize devices, handle paging, and timeouts gracefully.
- Record last successful sync and last error for banners/health.

## Error/Diagnostics
- Verbose logs on failures with request_id, UISP endpoint, status, and payload snippets (sanitized).
- Expose `/health` with lag and last error; emit events with correlation IDs.
