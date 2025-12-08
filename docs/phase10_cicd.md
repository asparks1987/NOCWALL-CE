# Phase 10 Item 5 - CI/CD Pipelines

## Pipelines
- Separate pipelines for api/poller/notifier/web/Android.
- Steps: lint/test/build/scan; push images; deploy to staging; gated promotion to prod.
- Android: signing, track promotion (internal/stage/prod), crash reporting.

## Error/Diagnostics
- Pipeline failures report with build ID and logs; banner can reference latest failure for operators.
