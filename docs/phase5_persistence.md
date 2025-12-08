# Phase 5 Item 4 - Metrics Persistence

## Behavior
- Write samples to Timescale hypertables; batch inserts; handle backpressure.
- Rollup jobs generate aggregates.

## Error/Diagnostics
- On write failure, log device_id/sample and error; expose backlog size and last error in health endpoint for banner.
