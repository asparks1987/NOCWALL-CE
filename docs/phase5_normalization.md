# Phase 5 Item 2 - UISP Payload Normalization

## Mapping
- Map UISP roles to internal roles (gateway/router/switch/ap/ptp).
- Extract identifiers (id/mac/name), IP, status, metrics.
- Handle missing/unknown fields with defaults and warnings.

## Error/Diagnostics
- On schema mismatches, log warnings with device_id and fields; banner shows “UISP data mismatch” for operators with request_id.
