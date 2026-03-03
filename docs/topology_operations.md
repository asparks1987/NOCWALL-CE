# Topology Operations Runbook

This runbook documents operational procedures for the topology graph introduced in the PRO topology phase. It covers safe rebuild and compaction workflows, validation checks, and troubleshooting guidance.

## Scope

- Inventory stitching and neighbor-link derived graph state in `api/store.go`.
- Topology-facing API endpoints:
  - `GET /topology/nodes`
  - `GET /topology/edges`
  - `GET /topology/health`
  - `GET /topology/path`
  - `GET /topology/ha/pairs`
  - `GET /topology/ha/events`

## Data Dependencies

Topology state is derived from:

- `device_identities`
- `neighbor_links`
- `source_observations` (indirectly via identity updates)

Because topology is generated from persisted facts, rebuild operations are deterministic for a given store snapshot.

## Rebuild Procedure

Use rebuild when topology output is stale, malformed, or after a migration/import that modifies identity or neighbor state.

1. **Take backup of store file**
   - Copy current persisted JSON store file before mutation/restart.
2. **Restart API process cleanly**
   - Graph is recomputed from persisted inventory at load and through new ingest events.
3. **Force representative ingest cycle**
   - Trigger pollers or ingest telemetry for each source to refresh neighbor/identity facts.
4. **Validate graph health**
   - Query `/topology/health` and confirm:
     - `node_count` and `edge_count` are non-zero for active environments.
     - `managed_node_count` aligns with expected managed device count.
     - `unknown_neighbor_edges` is within expected baseline.
5. **Spot-check path trace**
   - Query `/topology/path` for a known source/target pair.

## Compaction Guidance

Topology entities are not currently stored as an independent materialized table; compaction is performed by normal store maintenance and source-fact replacement logic.

Recommended periodic actions:

- Ensure source ingestion updates replace stale interface/neighbor facts for identical `(identity, source)` keys.
- Remove decommissioned devices via inventory cleanup so managed node count does not drift.
- Keep store snapshots and archive rotations to support rollback if a compaction pass removes needed facts.

If compaction-like cleanup is needed urgently:

1. Export store backup.
2. Prune obsolete inventory records offline.
3. Restart API to reload/prune topology derivations.
4. Validate with `/topology/health` and fixture-style smoke tests.

## Troubleshooting

### Symptom: missing expected edge between managed devices

Checks:

- Confirm both devices still exist in `device_identities` and have fresh observations.
- Confirm at least one side reports the peer in `neighbor_links`.
- Verify device names/hints allow identity resolution.

Remediation:

- Re-run source poll for both devices.
- If matching is ambiguous, provide stronger identity hints (MAC/serial/hostname consistency).

### Symptom: too many unresolved/external nodes

Checks:

- Inspect `unknown_neighbor_edges` in `/topology/health`.
- Review recent source payload quality for neighbor identifiers.

Remediation:

- Improve neighbor identity hints in upstream source adapters.
- Confirm naming conventions are stable and not duplicated across sites.

### Symptom: path trace returns not found

Checks:

- Verify source and target identities resolve to valid topology node IDs.
- Ensure graph has a connected path between node components.

Remediation:

- Validate with `/topology/nodes` and `/topology/edges` first.
- Rebuild topology via restart + ingest cycle.

## Validation Checklist (Post-Change)

- `go test ./...` passes for API packages.
- Topology fixture tests pass:
  - `TestTopologyFixtureTriangleResolvedPath`
  - `TestTopologyFixtureBranchIncludesUnknownNeighbors`
- `/topology/health` has sane counts in a representative tenant.
