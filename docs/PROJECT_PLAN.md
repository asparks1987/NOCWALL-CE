# PROJECT_PLAN (Consolidated Pointer)

Primary planning documents are now:
- `README.md` - current architecture direction, feature status, run/test commands.
- `BURNDOWN.md` - chronological multi-phase execution plan (10 epics).

This file remains as a consolidation pointer to reduce planning drift across dozens of phase markdown files.

## Canonical Direction
- Product name: **NOCWALL-CE**.
- Commercial model: hosted-first at `nocwall.com` with provisioned user workspaces.
- Data sources:
  - Vendor API ingestion (UISP and future connectors)
  - Local network agent installed on Linux/SBC
- Wallboard priority: high-density, glanceable device cards with alert-first UX.

## Doc Hygiene Rules
- Add new roadmap items to `BURNDOWN.md` only.
- Update current functionality/limitations in `README.md` only.
- Keep legacy phase files in `docs/` as historical context unless explicitly revived.
