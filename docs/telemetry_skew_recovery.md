# Telemetry Skew and Source Degradation Runbook (R24)

Operational runbook for recovering from telemetry clock skew and source degradation events.

## Scope

- Source clock skew causing corrected timestamps or low confidence.
- Source poll degradation (high poll failure rate, stale ingest, missing data).
- Active `telemetry_gap` incident spikes.

## Detection Signals

Use these APIs:

- `GET /telemetry/quality`
- `GET /telemetry/ingestion/health`
- `GET /telemetry/governor`
- `GET /telemetry/retention`

Key indicators:

- `scorecards[].warnings` includes `low_timestamp_confidence` or `stale_ingest`.
- `scorecards[].stats.clock_skew_violation_count` increases quickly.
- `health.poll_failures` / `health.poll_attempts` trends upward.
- `health.active_gap_incidents` is non-zero.

## Triage Checklist

1. Confirm blast radius:
   - Identify affected `source` values in scorecards.
   - Compare affected sources against sites/roles in the dashboard.
2. Verify clock correctness:
   - Check API host NTP sync.
   - Check source/agent host NTP sync and timezone configuration.
3. Validate source reachability:
   - Run source diagnostics from Account Settings.
   - Trigger `Poll Now` per source and inspect HTTP/error responses.
4. Validate ingest freshness:
   - Inspect `last_ingest_at_ms` and `freshness_score` in scorecards.
   - Confirm gaps correlate to real source outages vs clock offsets.

## Recovery Actions

### A. Clock Skew Recovery

1. Correct NTP on source/agent hosts.
2. Restart source agent or polling service after time correction.
3. Trigger manual poll.
4. Verify:
   - `timestamp_corrected_count` growth slows.
   - `low_timestamp_confidence` warning clears.
   - `clock_skew_violation_count` stabilizes.

### B. Source Degradation Recovery

1. Validate API token and endpoint for the degraded source.
2. Resolve network/TLS/DNS path issues from the API host.
3. Trigger manual poll and watch for successful attempts.
4. Verify:
   - `poll_failures` stops increasing.
   - `freshness_score` returns to healthy ranges.
   - `telemetry_gap` incidents auto-resolve as data resumes.

## Escalation

Escalate when any condition persists for >15 minutes:

- Poll failure rate >20% on a source.
- Consecutive poll failures >=3.
- Active gap incidents continue rising.
- Timestamp corrections continue rising after NTP fixes.

Include in escalation payload:

- Affected source IDs.
- Current scorecards and warnings.
- Poll failure/error snippets.
- Start time and actions already attempted.

## Post-Incident Validation

1. Confirm no active `telemetry_gap` incidents remain for recovered sources.
2. Confirm scorecards return to `healthy`/`degraded` and not `failing`.
3. Confirm dashboard data freshness normalizes.
4. Record root cause and preventive action in ops notes.
