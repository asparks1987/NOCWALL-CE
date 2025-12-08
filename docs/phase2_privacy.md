# Phase 2 Item 10 - Data Privacy, Secrets, PII Boundaries

## PII Boundaries
- Store minimal user data: name, email, role, optional phone. No UISP customer PII beyond device/site names.
- Tokenize device/user identifiers; avoid logging secrets or full payloads in plaintext.

## Secrets Handling
- Dev: env files (gitignored); Stage/Prod: secret store (K8s secrets/manager); rotate keys (OIDC, FCM, PAT signing).
- Encrypt at rest where available (Postgres disks, backups).

## Logging & Tracing
- Structured logs with request_id; redact tokens/passwords; sample traces carefully to avoid PII leakage.
- Verbose error banner should sanitize sensitive fields; include correlation ID not secrets.

## Retention
- Metrics retention as defined; audit logs retained per policy; allow configurable retention per tenant/site if needed.

## Compliance
- Note any geo/data residency constraints; ensure backups follow same region if required.
- Provide data export/delete for user records if compliance demands.
