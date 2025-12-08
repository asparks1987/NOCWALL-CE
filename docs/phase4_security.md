# Phase 4 Item 8 - Security Hardening (API)

## Controls
- Rate limiting per IP/user on auth and write endpoints.
- CSRF protection for cookie flows; same-site cookies; secure flags.
- Input validation on all params/bodies; reject unknown fields.
- CORS allowlist per environment.

## Headers
- Security headers (HSTS, CSP, X-Frame-Options, etc.) via Caddy/api.

## Error/Banner Expectations
- On security violations, respond with codes (`rate_limited`, `csrf_failed`, `validation_failed`) and request_id; banner shows retry guidance.
