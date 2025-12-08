# Phase 1 Item 5 - Identity, RBAC, Token Strategy

## Identity Provider & Flows
- Primary: OIDC/OAuth2 (e.g., Okta/Azure AD/Keycloak). Browser uses PKCE; Android uses PKCE + token storage; API supports client credentials for automation.
- Fallback: local accounts for dev/lab with password + session cookies.

## Roles (initial)
- `admin`: full control (policies, users, maintenance windows, alerts).
- `operator`: acknowledge/clear incidents, manage maintenance, view history.
- `viewer`: read-only dashboards/incidents/history.
- `wallboard`: read-only, kiosk-safe, auto-refresh; no actions.

## Tokens & Sessions
- Browser: short-lived access tokens + refresh token; HTTP-only cookies; CSRF protection for state-changing calls.
- API/machine: PAT/service tokens scoped to tenant/site and role; revocable; expiry required.
- Push/device: device tokens per platform stored with user ID and environment; rotate on logout.
- Include request/correlation IDs in responses for banner/debugging.

## Group/Claim Mapping
- Map IdP groups/claims to roles; support per-tenant overrides if multi-tenant emerges.
- Require email, name, unique subject; optional phone for SMS escalation.

## Security Hardening
- Rate limiting on auth endpoints; lockout/backoff on failed logins (local auth).
- Refresh rotation and reuse detection; session revocation on password change or logout.
- Signed/encrypted cookies in prod; same-site strict for browser session.

## Error/Banner Expectations
- Auth failures emit structured errors with code (`auth_invalid`, `auth_expired`, `auth_csrf`, `auth_oidc_failed`) plus correlation ID.
- UI/Android banner shows reason, next steps (retry/login), and “copy diagnostics”.
