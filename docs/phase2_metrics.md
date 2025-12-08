# Phase 2 Item 5 - Metrics Storage & Retention

## Store
- Postgres with Timescale extension for time-series metrics.
- Tables: device_metrics (latency/cpu/ram/temp/loss/jitter/throughput), site_aggregates, poller_stats.

## Retention
- Raw samples: 30-90 days (configurable) with compression.
- Rollups: 5m/1h/daily aggregates retained longer (6-12 months).
- Incidents/alerts retained per compliance; audit logs per policy.

## Indexing & Performance
- Index on (device_id, timestamp); hypertables for Timescale.
- Continuous aggregates for fast charts.

## Backups/Recovery
- Nightly logical backups; PITR recommended.
- Test restore flows; document RPO/RTO per environment.

## Error/Diagnostics
- Health endpoints expose write latency, backlog, and errors.
- Verbose errors on failed writes (including sample payload, device_id, request_id) for operator banners.
