# Telemetry Retention (R17)

NOCWALL now applies a tiered retention policy to `source_observations` during ingest and store initialization.

## Tier policy

- **Hot**: up to 24 hours old, keep all points.
- **Warm**: older than 24 hours and up to 7 days old, keep every 3rd point.
- **Cold**: older than 7 days and up to 30 days old, keep every 10th point.
- Data older than 30 days is dropped.

A hard cap of `maxSourceObservations` is still enforced after tier compaction.

## Diagnostics API

Use `GET /telemetry/retention` to inspect the latest retention run summary:

- `before_count`
- `after_count`
- `dropped_count`
- per-tier retained counts and sampling controls

This endpoint is intended for PRO telemetry validation and operational diagnostics.
