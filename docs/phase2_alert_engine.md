# Phase 2 Item 4 - Alert Engine Inputs/Outputs

## Inputs
- `device_state.changed` events (status, role, latency, metrics, upstream).
- `metric.sampled` rollups for latency/loss/jitter/utilization.
- Maintenance windows, dependency graph, user preferences (quiet hours), suppression rules.

## Evaluation
- Rule sets per role (see Phase 1 alert policies).
- Stateful detection for flaps, sustained latency/loss/jitter, CPU/RAM/temp, throughput.
- Suppression applied for maintenance and dependency; records suppression reason.

## Outputs
- `incident.triggered` with context (thresholds breached, samples, suppression applied?).
- `incident.updated` when state changes (acknowledged, suppressed, escalated).
- `incident.resolved` when cleared.
- Notifications scheduled (priority, channels) with correlation IDs.

## Error/Diagnostics
- On rule evaluation failures, emit diagnostic events with stack/error, correlation_id, device_id, and last samples.
- Logs include which rule matched and threshold values for operator banners.
