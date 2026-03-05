# Telemetry Alert Intelligence

This document covers the PRO alert-intelligence layer added for confidence scoring, impact radius estimation, and storm-shield summarization.

## API

- `GET /telemetry/alerts/intelligence`
- Query params:
  - `limit` (default `40`, max `120`)
  - `window_minutes` (default `15`, max `1440`)
  - `burst_threshold` (default `4`, max `100`)

## Response highlights

- `alerts[]`: active incidents enriched with:
  - `confidence_score` (`0.05` to `0.99`)
  - `confidence_level` (`high`, `medium`, `low`)
  - `confidence_reasons[]`
  - `impact` with managed reach, percent reach, and estimated scope (`local`, `site`, `network`)
- `storm_bursts[]`: grouped high-volume incident bursts within the active window.
- `raw_alert_count` and `summarized_alert_count`: direct comparison output for raw vs storm-shielded alert volume.

## Confidence scoring inputs

Current pipeline weighs:

- incident severity and type (`offline`, `telemetry_gap`, etc.)
- incident duration
- latest telemetry sample agreement (for example, offline incident plus latest sample still offline)
- timestamp confidence from ingest normalization
- source quality score trend
- device role weighting (core roles receive a small confidence boost)

## Impact radius estimator

Impact uses the live topology graph:

- finds the incident device node
- walks connected dependencies
- computes managed-node reach and percentage coverage
- classifies scope:
  - `local`
  - `site`
  - `network`

## UI exposure

In the topology tab:

- **Alert Confidence and Impact Radius** panel shows active incidents with confidence badges and impact scope.
- **Storm Shield Summary** panel shows burst groups and raw-vs-summarized reduction.

