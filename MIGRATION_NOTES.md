# MIGRATION_NOTES

This pass moves the project toward the NOCWALL-CE hosted-first model while keeping local development runnable.

## Completed in this pass
- Rebranded active dashboard surface from UISP NOC to NOCWALL-CE.
- Added dashboard display configurability (density + metric visibility) with persistent browser settings.
- Added API stubs for hosted model flow:
  - `GET /agents`
  - `POST /agents/register`
  - `POST /telemetry/ingest`
  - `POST /events/ingest`
- Wired ingest to in-memory incident creation/resolution for basic device up/down behavior.
- Fixed store mutation/save deadlock risk in API store methods.
- Removed hardcoded secrets from Compose and introduced `.env.example`.
- Consolidated planning docs around `README.md` and `BURNDOWN.md` and reduced `docs/PROJECT_PLAN.md` to a pointer.

## Intentionally left as stubs
- Multi-tenant SaaS control plane and provisioning.
- Production auth/RBAC/SSO.
- Production agent enrollment security lifecycle.
- Correlation/dedup/escalation/on-call routing.
- Billing/licensing and Pro-only enterprise workflows.
