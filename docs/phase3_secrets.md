# Phase 3 Item 3 - Secrets & Config Templates

## Dev
- `.env` files (gitignored) for UISP token, OIDC creds, FCM, Gotify; sample `.env.example`.
- Verbose logging enabled; secret scanning pre-commit.

## Staging/Prod
- K8s Secrets/secret manager; mount as env/volume; separate per environment.
- Rotation policy for OIDC client secrets, FCM keys, PAT signing keys.

## Templates
- Helm values templates for all services with placeholders for secrets and URLs.
- Compose override templates for contributors.

## Error/Diagnostics
- Startup checks fail fast with clear banner-friendly errors if secrets missing/invalid (which key, which service, next steps).
