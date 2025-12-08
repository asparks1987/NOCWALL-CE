# Phase 7 Item 5 - Deliverability Metrics & Dead Letter

## Metrics
- Track per-channel success/fail, latency, retries.
- Expose counts for banner (“notification failures detected”) with last error summary and request_id.

## Dead Letter
- Failed deliveries after retries go to DLQ with payload, error, and correlation_id.
- Admin UI to view/retry DLQ entries.
