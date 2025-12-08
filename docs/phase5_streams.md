# Phase 5 Item 3 - Event Publishing & Dedupe

## Publishing
- Emit `device_state.changed` and `metric.sampled` to Redis Streams with idempotency keys.
- Include correlation_id and request_id.

## Dedupe
- Skip publishing if state identical to previous snapshot within window; track hash per device.
- Configurable dedupe window to reduce noise.

## Error/Diagnostics
- On publish failure, log with stream name, device_id, and error; push to dead-letter if repeated.
