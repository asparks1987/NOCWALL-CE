# Codebase Inventory (Phase 1 / Item 1)

Inventory of current codepaths and components.

## PHP UI + Server
- `index.php`: Single-page UI and AJAX handler (devices, history, gotify test, TLS provisioning, auth, ack/simulate), SQLite init, Gotify send, TLS modal, cache handling.
- `cache/`: Runtime state (status cache, metrics.sqlite, auth.json, Gotify token/logs).

## Frontend Assets
- `assets/app.js`: Polling loop, siren/ack logic, badges, history loading, simulate outage, TLS modal wiring.
- `assets/device-detail.js`: Device modal rendering and charts.
- `assets/style.css`: Layout/styling for dashboard/tabs/modals.
- `buz.mp3`: Siren audio.

## Android (WebView Wrapper)
- `android/`: Kotlin WebView app; `MainActivity.kt` loads dashboard URL, caches creds; assets/icons; `android/README.md` for setup.

## Container, Startup, and Proxy
- `Dockerfile`: PHP 8.2 + Apache base with bundled assets and Gotify.
- `docker-compose.yml`: Services for `uisp-noc` and `caddy`, volumes for cache/certs, env vars for UISP, Gotify, TLS.
- `start.sh`: Boots Gotify, prepares config, then starts Apache.
- `buildmultiarch.sh`: Single command runner with `/install`, `/build`, `/update` actions for CE build/publish workflow.
- `Caddyfile`: Reverse proxy/TLS config; optional Gotify host.

## Embedded Gotify
- Bundled Gotify 2.6 launched from `start.sh`; config under `/etc/gotify`; auto-generates token in `cache/gotify_app_token.txt`; logs to `cache/gotify_log.txt`.
- `docs/GOTIFY.md`: Usage, defaults, external Gotify override.

## Documentation/Specs
- `README.md`: Project overview, current features, Android direction, roadmap link.
- `docs/`: CADDY, DOCKERHUB summary, GOTIFY, PROJECT_PLAN roadmap.
- `swagger.json`: UISP API reference snapshot (used for reference; not wired into code).

## TLS/Reverse Proxy
- `docs/CADDY.md`: TLS/ACME/Caddy usage; in-app provisioning notes.
- Caddy admin endpoints used by `index.php` when TLS UI is enabled.
