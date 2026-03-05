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
