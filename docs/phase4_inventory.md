# Phase 4 Item 3 - Inventory Endpoints

## Endpoints
- `GET /devices` with filters (site, role, status, tags, search); pagination.
- `GET /devices/{id}` detail including upstream, last metrics summary.
- `PATCH /devices/{id}` for tags/labels/manual upstream override.
- `GET /sites`, `GET /sites/{id}`, `POST/PATCH /sites` (admin).
- `GET /topology` for dependency graph with validation status.

## Error/Banner Expectations
- Validation errors list invalid fields; include request_id.
- On missing upstream/topology issues, return warning details to show in banner.
