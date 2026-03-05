# Telemetry Baselines and Anomaly Windows

This document describes the dynamic telemetry baseline model and anomaly windows used in PRO topology/telemetry views.

## Overview

- Baselines are computed from retained telemetry samples across hot/warm/cold tiers.
- Data is grouped by:
  - device role
  - site
- Metric baselines currently include:
  - `latency_ms`
  - `availability_pct`

## Dynamic Baselines

For each role/site group, the API computes:

- sample count
- mean
- standard deviation
- p50
- p95
- dynamic lower/upper bounds (`mean ± 2*sigma`, clamped by metric domain)

Response endpoint:

- `GET /telemetry/baselines`
- Optional query param: `window_hours` (default `336` / 14 days, capped at 90 days)

## Anomaly Windows

The same baseline response includes day/hour windows:

- day of week (UTC)
- hour of day (UTC)
- sample count
- latency mean/stddev
- availability mean/stddev

These windows support seasonality-aware anomaly workflows by exposing expected behavior for each day/hour bucket.

## UI Exposure

In the topology tab telemetry section, operators can review:

- role/site baseline groups with metric bounds
- day/hour anomaly windows (sample count + expected means)

## Notes

- Baselines require sufficient telemetry density (`minBaselineSamples`) to avoid unstable windows.
- This implementation is intended as a reliability-focused foundation for later alert confidence and storm-shield features.
