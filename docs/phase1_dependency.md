# Phase 1 Item 8 - Dependency Mapping & Suppression

## Topology Model
- Sites contain gateways; gateways connect to APs/switches; APs may serve downstream devices.
- Store parent/child relationships to enable suppression and blast-radius views.

## Suppression Rules
- If gateway offline, suppress downstream AP/switch alerts; show “suppressed due to upstream” badge with upstream ID.
- If maintenance on parent, inherit maintenance for children.
- Logs must record suppression reason, parent device, and correlation ID for debugging.

## Detection & Sync
- Ingest topology from UISP where available; allow manual overrides/tags in UI for missing links.
- Validate topology on ingest; log inconsistencies with verbose error messages for operators.

## UI/UX
- Visual indicator when alerts are suppressed; provide “force alert” toggle for testing.
- Dependency graph view for wallboard and device detail.

## Error/Banner Expectations
- When suppression occurs, banner note includes parent state, time started, and when alerts will resume.
- On topology fetch errors, display verbose banner with missing data, request IDs, and suggested fix (e.g., resync UISP or set manual link).
