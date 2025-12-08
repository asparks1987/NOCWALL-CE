# Phase 1 Item 9 - Migration Strategy (Legacy PHP/WebView -> New Stack)

## Approach
- Parallel run: stand up new API/poller/notifier + SPA/Android alongside legacy PHP UI/WebView.
- Mirror polling and alerts; compare state/alerts to ensure parity before cutover.
- Feature flags to toggle UI between legacy and new SPA; keep legacy available until validation complete.

## Stages
1. Stand up staging stack with mirrored polling and notifications.
2. Validate device state parity and alert parity; log divergences with verbose diagnostics (device ID, role, sample data, timestamps).
3. Pilot SPA with subset of users; pilot native Android with on-call techs.
4. Train/runbooks; collect feedback; fix gaps.
5. Cutover window: enable new stack in prod, keep legacy read-only for fallback.
6. Monitor and be ready to rollback; document rollback steps.
7. Deprecate legacy PHP UI/WebView after stable period; archive images and docs.

## Error/Banner Expectations During Migration
- SPA/Android show banner when using legacy vs new backend, including environment and feature flag state.
- If parity check detects mismatches, surface banner to admins with counts and link to diagnostics.
- Provide explicit “using legacy” vs “using new stack” indicator for operators.
