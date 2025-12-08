# Phase 5 Item 8 - Poller Edge Case Tests

## Cases
- UISP timeout/error; auth failure; partial data; malformed payload; large device counts.
- Backoff/retry logic; dedupe correctness; missing upstream links.

## Expectations
- Tests assert verbose error logging with request_id and banner-ready messages.
- Health endpoint reflects failure and recovery.
