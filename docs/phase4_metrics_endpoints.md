# Phase 4 Item 5 - History/Metrics Endpoints

## Endpoints
- `GET /metrics/devices/{id}` with time range, resolution, rollups.
- `GET /metrics/sites/{id}/aggregate` for site-level health.
- `GET /health/poller` for lag and last error (for banners).

## Behavior
- Paginate/stream large ranges; support downsampling.
- Include last sample and summary stats for device detail.

## Error/Banner Expectations
- Errors include request_id and hints (invalid range, missing device, backend unavailable).
