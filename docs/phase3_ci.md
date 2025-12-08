# Phase 3 Item 5 - Migrations Tooling & CI Hooks

## Migrations
- Tooling: Prisma/Flyway/Liquibase (pick based on API stack); versioned migrations for Postgres/Timescale.
- Migration runner as part of api start or separate job; idempotent; fails fast with verbose error output.

## CI Hooks
- CI steps: lint -> test -> build -> run migrations against ephemeral DB -> package images.
- Migration lint/check for drift; generate schema diagram artifact.
- CI surfaces errors with full logs and summaries suitable for banner “migration failed” notices (with run ID).

## Rollback
- Document rollback steps for failed migrations; backups before production runs.
