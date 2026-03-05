# Telemetry Reliability Layer (R17-R24)

NOCWALL now applies retention, sampling, clock-skew normalization, quality scoring, and missing-signal controls for telemetry reliability.

## Retention tier policy (R17)

- **Hot**: up to 24 hours old, keep all points.
- **Warm**: older than 24 hours and up to 7 days old, keep every 3rd point.
- **Cold**: older than 7 days and up to 30 days old, keep every 10th point.
- Data older than 30 days is dropped.

A hard cap of `maxSourceObservations` is still enforced after tier compaction.

## Sampling governor and queue priority (R18)

Telemetry ingest now applies per-device-class sampling intervals and poll-queue priorities:

- `core`: 5s minimum sample interval, queue priority 0
- `distribution`: 10s minimum sample interval, queue priority 1
- `access`: 15s minimum sample interval, queue priority 2
- `edge`: 30s minimum sample interval, queue priority 3
- `default`: 20s minimum sample interval, queue priority 4

State transitions (`device_down`, `offline`, `device_up`, `online`) bypass sample suppression.
Polled source events are ingested in queue-priority order before persistence.

## Gap detector (R19)

NOCWALL now detects missing telemetry signals and manages `telemetry_gap` incidents:

- Gap threshold is class interval `x4` (minimum 2 minutes).
- Detector creates unresolved `telemetry_gap` incidents when stale.
- Detector resolves active `telemetry_gap` incidents when signal freshness recovers.
- Gap detection runs during telemetry ingest and during source poller loops.

## Clock skew normalization (R20)

Telemetry ingest accepts optional source timestamps:

- `observed_at_ms`
- `observed_at` (RFC3339)

Ingest applies source clock-skew checks and normalizes the effective observation timestamp:

- Future skew beyond 2 minutes is corrected to ingest time.
- Source timestamps older than 7 days are corrected to ingest time.
- Confidence (`0.0-1.0`) is computed from absolute skew.
- Normalized metadata is persisted on observations and telemetry samples:
  - `source_observed_at`
  - `clock_skew_ms`
  - `timestamp_confidence`
  - `timestamp_corrected`

## Source quality scorecards (R21)

NOCWALL computes per-source quality scorecards from ingest + poll outcomes:

- Freshness score
- Completeness score
- Poll error rate
- Skew average/max with low-confidence and corrected timestamp counts

Scorecards are classified into `healthy`, `degraded`, or `failing`.

## API and UI views (R22)

API:

- `GET /telemetry/quality`
- `GET /telemetry/ingestion/health`
- `GET /telemetry/governor`
- `GET /telemetry/retention`

UI:

- PRO topology tab now renders source quality scorecards and ingestion health summaries using `?ajax=telemetry_quality`.

## Load tests (R23)

Benchmarks added in `api/store_benchmark_test.go`:

- `BenchmarkTelemetryRetentionCompaction`
- `BenchmarkTelemetrySamplingGovernorIngest`

Run:

```bash
go test -run ^$ -bench BenchmarkTelemetry -benchmem ./...
```

## Diagnostics APIs

Use `GET /telemetry/retention` to inspect the latest retention run summary:

- `before_count`
- `after_count`
- `dropped_count`
- per-tier retained counts and sampling controls

Use `GET /telemetry/governor` to inspect the current governor state:

- `accepted_samples`
- `dropped_samples`
- `active_gap_incidents`
- class rules with interval and queue priority

These endpoints are intended for PRO telemetry validation and operational diagnostics.
