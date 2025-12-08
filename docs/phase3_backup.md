# Phase 3 Item 8 - Backup & Restore

## Backups
- Postgres: nightly logical backups + optional PITR; store encrypted; retain per policy.
- Redis: not primary store; snapshot if needed for queued events (optional).
- Gotify (if used): include in Postgres or separate DB backup.
- Caddy/ACME data: back up `caddy_data` if using standalone; otherwise reissue certs.

## Restore
- Document restore steps for staging/prod; test restores regularly.
- Validate data integrity post-restore (counts, sample queries).

## Error/Diagnostics
- Backup jobs emit success/failure with request/run ID; banners for admins if backups fail.
