# Phase 1 Item 2 - Targets & Requirements (Draft)

Working draft to capture target users, roles, on-call schedule considerations, notification channels, and SSO needs. Update as decisions are confirmed.

## Target Users & Roles (proposed)
- Field/on-call technician: Needs actionable alerts, acknowledges, maintenance toggles, device detail, and mobile-friendly UI.


## On-Call & Scheduling (proposed)
- Quiet hours/override windows per user/team.

## Notification Channels (initial set)
- FCM push (Android native + browser push via service worker).


## Identity / SSO (proposed)

- Token strategy: short-lived access tokens with refresh; device tokens for push registration.

## Open Items to Confirm
- Which IdP(s) to support first and required claims (group/role mapping).
- Minimum channels required for GA vs. nice-to-have (e.g., SMS vs. webhooks).
    just push is fine
- Rotation source of truth (built-in schedule vs. external paging tool).
built in
- Per-tenant/multi-site needs (if multi-tenant required).
per tenent
- Compliance constraints (PII/log retention, geo residency).
no need
