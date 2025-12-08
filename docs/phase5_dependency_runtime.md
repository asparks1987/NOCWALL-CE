# Phase 5 Item 5 - Dependency Awareness in Poller

## Behavior
- Poller tags devices with upstream info (from UISP or manual overrides).
- Emits events including upstream_id to enable suppression.

## Error/Diagnostics
- On missing/ambiguous upstream, emit warning with device_id and suggested fix; health endpoint counts unresolved links for banner.
