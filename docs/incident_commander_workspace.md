# Incident Commander Workspace

This document describes the Phase 3 incident commander workflow added for `R33` and `R39`.

## Scope

- Commander ownership assignment on active incidents
- Command timeline entries for incident lifecycle and operator notes
- Incident workspace view in the topology tab
- Shift handoff brief generation with delta summaries
- Incident audit event stream for commander handoff and checklist actions

## API Endpoints

All endpoints require API auth (`Authorization: Bearer ...` when enabled).

- `GET /incidents/workspace`
  - Query params:
    - `active_limit` (default `80`, max `400`)
    - `recent_limit` (default `40`, max `400`)
  - Returns active and recent incidents plus assignment counts.

- `POST /incidents/:id/commander`
  - Body:
    - `commander` string (empty string clears ownership)
    - `actor` string (optional, operator identity)
  - Adds timeline events:
    - `commander_assigned`
    - `commander_cleared`

- `POST /incidents/:id/timeline`
  - Body:
    - `event_type` string (defaults to `note`)
    - `message` string (required)
    - `actor` string (optional)
  - Appends timeline entry to the incident.

- `GET /incidents/handoffs`
  - Query params:
    - `limit` (default `30`, max `200`)
  - Returns most-recent-first shift handoff brief history.

- `POST /incidents/handoff/generate`
  - Body:
    - `actor` string (optional)
    - `note` string (optional)
    - `active_limit` int (optional)
  - Generates a handoff brief with:
    - active/unassigned counts
    - new-active deltas since prior handoff
    - resolved-since-last deltas
    - commander-change deltas

- `GET /incidents/audit`
  - Query params:
    - `limit` (default `120`)
    - `incident_id` (optional)
    - `action` (optional)
  - Returns audit events for incident operations.

- `POST /incidents/:id/checklist/audit`
  - Body:
    - `checklist_id`, `step_id`, `state`, `note`, `actor`
  - Records a checklist audit event and appends a timeline note.

## Automatic Timeline Events

The store now emits timeline entries for:

- Incident opened (`opened`)
- Incident acknowledged (`acked`)
- Incident resolved (`resolved`)
- Commander assign/clear (`commander_assigned`, `commander_cleared`)
- Operator note (`note`)

Existing incidents without timeline history are backfilled with an initial `opened` event on load/migration.

## PHP Bridge Actions

UI-facing AJAX actions in `app.php`:

- `?ajax=incident_workspace`
- `?ajax=incident_commander_assign` (`POST`, CSRF required)
- `?ajax=incident_timeline_add` (`POST`, CSRF required)
- `?ajax=incident_handoff_history`
- `?ajax=incident_handoff_generate` (`POST`, CSRF required)
- `?ajax=incident_audit_events`
- `?ajax=incident_checklist_audit_add` (`POST`, CSRF required)

These actions proxy to the API and include the current session username as timeline actor.

## UI Behavior

When PRO incident workspace flags are enabled, topology tab includes:

- Active incident commander workspace panel
- Commander controls:
  - Claim (self-assign)
  - Assign (username input)
  - Clear
- Timeline note input per active incident
- Recent incident timeline panel
- Shift handoff generator controls and brief history
- Incident audit events list

## Validation

Suggested checks:

```bash
node --check assets/app.js
php -l app.php
(cd api && go test ./...)
```
