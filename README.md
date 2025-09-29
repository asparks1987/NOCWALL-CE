<div align="center">

# UISP NOC

Monitoring that feels crisp, clear, and loud when it matters.

</div>

---

What Is This?

UISP NOC is a self-hosted, lightweight Network Operations Center (NOC) dashboard for Ubiquiti UISP environments. It runs as a single Dockerized PHP + Apache app, stores short-term metrics in SQLite, and can push notifications via an embedded Gotify server. Caddy is included to front the app and handle HTTPS.

---

Feature Highlights

- Live UISP Polling: Queries UISP every ~10s for device status and metrics.
- Gateways + CPEs: Two tabs with device cards and at-a-glance health.
- Offline-First Sorting (CPE): CPE list surfaces offline devices first, then A–Z.
- Siren Alerts + ACK: Audible siren for unacknowledged offline gateways; quick 30m/1h/6h/8h/12h acknowledgements.
- Push Notifications: Embedded Gotify server for OFFLINE/ONLINE events (auto-bootstraps an app token).
- History: SQLite time-series (gateway CPU/RAM/Temp/Latency) with charts.
- Simple Sign-On (App Login): Built-in login with default admin/admin and UI password change; credentials persist in the data volume.
- Caddy Reverse Proxy: Ships with a Caddy service for HTTPS termination and routing to the app and optional Gotify.

---

Quick Start

1) Configure and launch (Docker Compose)

```bash
docker compose up -d
```

Environment defaults come from `docker-compose.yml`. Set at minimum:

- `UISP_URL`: e.g., `https://your-uisp.unmsapp.com`
- `UISP_TOKEN`: UISP API token

2) Open the UI

- With Caddy: `https://<NOC_DOMAIN>/` (or `http://localhost` if using local certs)
- Sign in with default credentials: admin / admin
- Change your password from the header once logged in

3) Embedded Gotify (optional)

- The container autostarts Gotify and provisions an application token. The app reads it from `cache/gotify_app_token.txt`.
- Exposing Gotify publicly is optional; see TLS/Gotify notes below.

---

Sign-On (Current)

- Default username: `admin`
- Default password: `admin`
- Change Password: Use the “Change Password” button in the header.
- Storage: Credentials are stored in `cache/auth.json` (inside the `noc_cache` volume).
- Reset: Stop the container and remove `cache/auth.json` to reset to admin/admin.

---

TLS / Let’s Encrypt (Current)

- How it works: Caddy terminates TLS for the UI (and optionally Gotify). The Caddy admin API is enabled on the internal Docker network so the app can request config reloads and trigger certificate provisioning.
- UI Provisioning: A “TLS/Certs” button exists but is hidden by default. You can enable it by setting `SHOW_TLS_UI=1` on the `uisp-noc` service.
  - The modal collects domain(s) and ACME email, then loads a live Caddy config via the admin API to start ACME issuance.
  - DNS for the chosen domain(s) must resolve to your host and ports 80/443 must be open to the internet.
- Manual provisioning: See `docs/CADDY.md` for details on domains and how Caddy integrates here.
- Current project: Finish and harden the end-to-end UX for HTTPS issuance (validation hints, staging mode guidance, better status feedback). For now, the UI toggle is off by default.

---

Configuration

App Environment (service `uisp-noc`)

- `UISP_URL` (required): Base URL of your UISP (e.g., `https://your-uisp.unmsapp.com`).
- `UISP_TOKEN` (required): UISP API token.
- `GOTIFY_DEFAULTUSER_NAME` (optional): Initial Gotify admin username (`admin` by default).
- `GOTIFY_DEFAULTUSER_PASS` (optional): Initial Gotify admin password (`changeme` by default — change on first run).
- `GOTIFY_URL` (optional): Override Gotify endpoint (defaults to embedded `http://127.0.0.1:18080`).
- `GOTIFY_TOKEN` (optional): If you manage tokens yourself, place the application token here; otherwise it will be auto-provisioned and saved to `cache/gotify_app_token.txt`.
- `SHOW_TLS_UI` (optional): `1/true/yes` to show the TLS/Certs button in the header.

Caddy Environment (service `caddy`)

- `NOC_DOMAIN`: Public hostname for the UISP NOC UI (e.g., `noc.example.com`). Use `localhost` for local development (Caddy issues local certs).
- `GOTIFY_DOMAIN` (optional): A second hostname for embedded Gotify (e.g., `gotify.example.com`). If omitted, Gotify is internal-only.
- `ACME_EMAIL`: Email used by Let’s Encrypt/ACME for certificate events.

Ports

- `caddy` maps `80:80` and `443:443` for HTTP/HTTPS. The `uisp-noc` container does not expose host ports by default (Caddy proxies to it). You can bypass Caddy by uncommenting the `ports` on `uisp-noc` in `docker-compose.yml` if desired.

---

Data & Persistence

Volumes

- `noc_cache:/var/www/html/cache` — application data:
  - `status_cache.json` — app runtime state (ACKs, ping metadata)
  - `metrics.sqlite` — gateway history data
  - `gotify/` — embedded Gotify data
  - `gotify_app_token.txt` — auto-provisioned Gotify app token used by the app
  - `auth.json` — app sign-on credentials (username + hashed password)
- `caddy_data:/data` — certificate storage (managed by Caddy)
- `caddy_config:/config` — Caddy runtime/config state

---

Troubleshooting

- No HTTPS certificate
  - Ensure DNS A/AAAA for `NOC_DOMAIN` points to your host and ports 80/443 are reachable from the internet.
  - Use the TLS UI with “Staging” enabled to test issuance without rate limits (`SHOW_TLS_UI=1`).
  - View current Caddy config via the TLS modal (or `docs/CADDY.md` for manual steps).

- Can’t log in
  - Reset by removing `cache/auth.json` (inside the `noc_cache` volume). The app will recreate default admin/admin on next start.

- No notifications
  - Make sure an application exists in Gotify and that the app has a valid token (auto-bootstrap writes it to `cache/gotify_app_token.txt`).

- History charts empty
  - History accumulates from per-minute gateway metrics. Give it a few minutes.

---

Roadmap / Current Work

- TLS UX polish: clearer validation hints, issuance status, and staging-first workflow. Button is hidden by default until this is complete.
- Optional OIDC SSO: integration with providers (Authentik/Authelia/Google/Okta) behind Caddy while keeping simple app login available.
- Additional device insights and charting improvements.

---

Development

- Stack: PHP 8.2 + Apache, SQLite, Vanilla JS + Chart.js, Caddy 2
- Layout:
  - `index.php` — backend + HTML template + API proxy + auth
  - `assets/` — JS, CSS
  - `cache/` — runtime state and databases (created on first run)
  - `start.sh` — launches embedded Gotify then Apache
  - `Caddyfile` — Caddy config for TLS and reverse proxy
  - `docs/CADDY.md` — Caddy/TLS notes and UI provisioning details

---

License

MIT — free to use, modify, and share.

