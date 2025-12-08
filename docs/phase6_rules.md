# Phase 6 Item 1 - Rule Evaluation Worker

## Scope
- Evaluate downtime, flaps, latency/loss/jitter, utilization, temperature.
- Uses events/metrics stream and suppression context.

## Error/Diagnostics
- On evaluation failure, emit diagnostic event with rule id, device_id, samples, error, correlation_id.
- Health reports last rule error and time for banners.
