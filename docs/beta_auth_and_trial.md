# Beta Auth and Trial Readiness

This document captures the beta-focused auth and account entitlements hardening shipped in the web app.

## Stripe-Managed Trial and Tiers

- New accounts are created with default CE inactive subscription state.
- Trial duration and activation are expected to be managed in Stripe (payment link/checkout configuration), not by local server assumptions.
- Billing subscribe supports Stripe payment links by tier:
  - `NOCWALL_STRIPE_PAYMENT_LINK` for single default checkout
  - `NOCWALL_STRIPE_PAYMENT_LINKS` for multi-tier checkout map (`{"pro_monthly":"https://...","pro_annual":"https://..."}`)
  - `NOCWALL_BILLING_DEFAULT_TIER` to pick the default tier when no explicit tier is selected
- Keep Stripe webhooks configured (`NOCWALL_STRIPE_WEBHOOK_SECRET`) so paid/trial state is synchronized back to account entitlements.
- A cleanup migration resets legacy local trial states (`notes=signup_trial_30d_free`) back to CE default so Stripe is the source of truth.

## Closed Beta Access Keys (Temporary PRO Entitlement)

- CE remains open/free; beta keys are optional and grant temporary PRO entitlement during closed beta.
- Key schema persisted in `cache/users.json` under `beta_keys`:
  - `code`, `status`, `max_redemptions`, `redeemed_count`, `expires_at`, `issued_by`, `notes`, `created_at`, `updated_at`
  - `redemptions` map of `username -> redeemed_at`
- Account-level key linkage is stored under `users.<username>.beta_access`.
- Entitlement precedence is:
  1. active paid/trial subscription
  2. valid redeemed beta key
  3. CE free
- Key audit events are stored in `beta_key_audit` for issue/redeem/revoke actions.
- Signup supports optional key redemption (`Create account` form `beta_key` field), and existing users can redeem from `Account Settings`.
- Ops CLI for key lifecycle:
  - `php scripts/beta_keys.php generate ...`
  - `php scripts/beta_keys.php list`
  - `php scripts/beta_keys.php disable --code ...`
  - `php scripts/beta_keys.php enable --code ...`

## Closed Beta v1 Smoke Validation

- Run `scripts/closed-beta-v1-smoke.sh` to validate:
  - account create/login/session flow
  - dashboard load baseline for wallboard usage
  - settings load + preference persistence path
  - CE entitlement baseline without beta key
  - multi-source onboarding from settings (`UISP`, `Cisco`, `Juniper`, `Meraki`, `Generic HTTP`) with save/test/diagnostics checks
  - live wallboard device ingestion checks for router/gateway monitoring across Cisco, Juniper, Meraki, and Generic HTTP sources
- `CB4` source onboarding/diagnostics validation supports two modes:
  - default local mock mode (`MOCK_UISP=1`): starts a disposable UISP-like endpoint in Docker and validates source save/test/diagnostics without external credentials
  - live mode: provide `UISP_TEST_URL` + `UISP_TEST_TOKEN` to validate against a real UISP instance
  - optional mock URL overrides:
    - `MOCK_UISP_SOURCE_URL` (default `http://host.docker.internal:18091`) used for the source saved into NOCWALL
    - `MOCK_UISP_HEALTH_URL` (default `http://127.0.0.1:18091`) used for host-side readiness checks
- `CBC8` connector UI onboarding validation supports:
  - `CISCO_TEST_URL` + `CISCO_TEST_TOKEN` for live Cisco checks
  - `JUNIPER_TEST_URL` + `JUNIPER_TEST_TOKEN` for live Juniper checks
  - when unset and mock mode is enabled, the script reuses local mock endpoints for deterministic coverage
- `CBC9` vendor-agnostic Generic HTTP validation supports:
  - `GENERIC_TEST_URL` for a live vendor-agnostic JSON endpoint
  - `GENERIC_TEST_TOKEN` when the endpoint requires auth
  - `GENERIC_TEST_API_PATH` (default `/feeds/generic/devices`) to point at a non-standard devices path
  - `GENERIC_TEST_AUTH_SCHEME` (`bearer`, `x-auth-token`, `token`, `authorization`, `none`)
  - when unset and mock mode is enabled, the script reuses a local mock JSON feed and confirms those devices appear in `?ajax=devices`
- `CBC10` Meraki validation supports:
  - `MERAKI_TEST_URL` using an organization-scoped base URL such as `https://api.meraki.com/api/v1/organizations/<organizationId>`
  - `MERAKI_TEST_TOKEN` for the Dashboard API key
  - when unset and mock mode is enabled, the script reuses a local mock Meraki status feed and confirms those devices appear in `?ajax=devices`

Example:

```bash
BASE_URL=http://localhost/app bash scripts/closed-beta-v1-smoke.sh
```

Live UISP example:

```bash
BASE_URL=http://localhost/app UISP_TEST_URL=https://example.unmsapp.com UISP_TEST_TOKEN=token_here bash scripts/closed-beta-v1-smoke.sh
```

## Session and Login Security Hardening

- Session cookie settings are explicitly hardened:
  - `HttpOnly=true`
  - `SameSite=Lax`
  - `Secure=true` when HTTPS is detected
  - strict session mode enabled
- Session ID is regenerated on successful login and registration.
- Logout clears the session cookie and destroys the session.

## CSRF Protection

- CSRF token is session-backed and generated with cryptographic randomness.
- Login/register forms include hidden `_csrf` values.
- Logout requires a valid CSRF token.
- Mutating AJAX routes require a valid CSRF token.
- Frontend bootstrap wraps `fetch()` for same-origin requests and automatically adds `X-CSRF-Token`.

## Login Rate Limiting

- Failed login attempts are tracked in `cache/login_attempts.json`.
- Lockout policy:
  - up to 8 failed attempts
  - rolling 15-minute window
  - 15-minute lockout when threshold is exceeded
- Tracking keys include:
  - IP-level key
  - username+IP key
- Successful login clears counters for the active username/IP.
