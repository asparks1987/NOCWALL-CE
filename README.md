# UISP NOC

A self-hosted, zero-friction Network Operations Center for Ubiquiti UISP deployments. UISP NOC polls your UISP controller every few seconds, persists short-term history in SQLite, sounds a siren for unacknowledged outages, and can push offline/online, flapping, and latency alerts through an embedded Gotify server. Caddy is bundled to front the app with HTTPS and to expose the optional Gotify UI on its own hostname.

---

## Why UISP NOC?

* **Purpose-built dashboard** – Gateways, routers/switches, and CPEs have dedicated grids with status badges, uptime, latency, CPU/RAM/temperature, and outage timers.
* **Actionable alerting** – Audible siren, one-click acknowledgements (30m → 12h), outage simulation for drills, and real-time health summaries.
* **Integrated notifications** – Gotify is shipped in the container, auto-provisioned, and used for offline/online, flapping, and sustained latency events. Bring your own Gotify instance by pointing the app at an external endpoint.
* **TLS automation** – A Caddy sidecar terminates TLS. Toggle the in-app provisioning modal (SHOW_TLS_UI=1) to request Let’s Encrypt certificates via the Caddy admin API.
* **Batteries included** – PHP 8.2 + Apache, Vanilla JS UI, Chart.js history views, siren media, SQLite metrics store, start script, and Docker Compose wiring.
* **Offline-friendly data** – Metrics and credentials live in a Docker volume; alerts, ping cache, and Gotify data are persisted across upgrades.

---

## Architecture Overview

```
┌──────────────┐     HTTPS (80/443)     ┌──────────────────────┐
│   Internet   │ ─────────────────────▶ │        Caddy          │
└──────────────┘                        │  • TLS termination    │
                                        │  • Reverse proxy      │
                                        │  • Optional Gotify    │
                                        └────────┬──────────────┘
                                                 │
                                         HTTP    │
                                          80     ▼
                                        ┌────────────────────────┐
                                        │      UISP NOC app      │
                                        │  • PHP 8.2 + Apache    │
                                        │  • UISP polling        │
                                        │  • SQLite history      │
                                        │  • Simple sign-on      │
                                        │  • REST-ish endpoints  │
                                        └────────┬───────────────┘
                                                 │
                                   localhost:18080│
                                                 ▼
                                        ┌────────────────────────┐
                                        │   Embedded Gotify      │
                                        │  • Alert delivery      │
                                        │  • Token bootstrap     │
                                        └────────────────────────┘
```

* `start.sh` ensures Gotify launches before Apache and writes configuration under `/etc/gotify`.
* `cache/` (mounted as `noc_cache`) stores credentials, runtime cache, SQLite `metrics.sqlite`, Gotify data, and log files.
* `assets/app.js` handles polling, siren logic, acknowledgement timers, device simulation, chart loading, and TLS modal interactions.
* `index.php` serves both the UI and AJAX endpoints (`devices`, `history`, `gotifytest`, `provision_tls`, etc.) and proxies the UISP API.

---

## Feature Deep-Dive

### Dashboard & UX

* Gateways, routers/switches, and CPEs are rendered in separate tabs with offline-first sorting.
* Aggregate health bar shows online counts, network health %, unacknowledged outages, average latency, and high CPU/RAM tallies.
* Device cards expose: reachability, uptime badge, outage duration, CPU/RAM/temp, current ping latency, and alert badges (flapping/latency).
* A history modal charts 24 hours of CPU, RAM, temperature, and ping metrics sourced from SQLite.
* "Simulate Outage" temporarily forces a device offline for drills and resets automatically.
* Acknowledge buttons silence the siren for preset durations, with a global "Clear All" shortcut.
* Optional TLS modal (hidden unless `SHOW_TLS_UI=1`) lets administrators load the current Caddy config, submit new domains, and toggle Let’s Encrypt staging mode.

### Polling & Metrics

* UISP is polled ~every 5 seconds (fast retry on changes, exponential backoff on errors).
* Gateways/routers/switches are pinged every minute; 10 random CPEs are spot-checked every three minutes, avoiding repeat pings within an hour.
* Metrics are appended to SQLite once per minute for backbone devices and exposed to the browser via `/index.php?ajax=history&id=<device_id>`.
* Automatic detection of sustained latency (`>=200 ms` for 3 consecutive samples) and flapping (>=3 transitions in 15 minutes) with Gotify alerts and badge indicators.

### Notifications

* Embedded Gotify listens on port 18080 (internal). The container auto-generates `/var/www/html/cache/gotify_app_token.txt` on first boot and logs delivery attempts to `cache/gotify_log.txt`.
* Offline notifications trigger after 30s of downtime and repeat every 10 minutes until recovery; online recovery alerts are sent immediately once back up.
* Flap and latency alerts respect per-device suppression windows; acknowledgements suppress audible siren but not Gotify messages.
* Use the “Send Test Notification” button (Settings menu) to validate connectivity.

### Authentication & Security

* Simple sign-on with credentials stored in `cache/auth.json`; default `admin/admin` is created on first run.
* Password change dialog enforces a minimum length and updates the stored hash atomically.
* Session-based login protects all AJAX endpoints; unauthenticated requests are bounced with HTTP 401.
* Caddy serves as TLS terminator and reverse proxy. Provisioning via the TLS modal uses Caddy’s admin API on the internal network (`caddy:2019`).

### Mobile Companion

* `android/` contains a Kotlin WebView wrapper that loads the dashboard on Android devices. It consumes `?ajax=mobile_config` to retrieve the UISP URL/token pair when running on the same network.
* The wrapper enables JavaScript & DOM storage and defaults to `http://10.0.2.2/` for emulator testing.

---

## Quick Start

1. **Clone & configure**

   ```bash
   git clone https://github.com/UISP-NOC/UISP-NOC.git
   cd UISP-NOC
   cp docker-compose.yml docker-compose.override.yml  # optional
   # edit environment values (UISP_URL, UISP_TOKEN, domains)
   ```

2. **Launch the stack**

   ```bash
   docker compose up -d
   ```

3. **Sign in**

   * Visit `https://<NOC_DOMAIN>/` (or `http://localhost` if developing locally).
   * Login with `admin / admin` and immediately change the password via the header menu.

4. **Wire notifications (optional)**

   * Use the embedded Gotify token at `cache/gotify_app_token.txt`, or set `GOTIFY_TOKEN` to a token from an external Gotify server.
   * Click **Settings → Send Test Notification** to confirm delivery.

5. **Enable TLS provisioning (optional)**

   * Set `SHOW_TLS_UI=1` on the `uisp-noc` service and restart.
   * Open the TLS modal, enter your domain(s) and ACME email, optionally toggle staging, and submit. Caddy will fetch certificates and reload.

---

## Configuration Reference

### `uisp-noc` service

| Variable | Required | Description |
| --- | --- | --- |
| `UISP_URL` | ✅ | Base URL to your UISP controller (e.g., `https://your-uisp.unmsapp.com`). |
| `UISP_TOKEN` | ✅ | UISP API token with device-read permissions. |
| `SHOW_TLS_UI` | ❌ | `1/true/yes` to expose the TLS provisioning modal in the UI. |
| `GOTIFY_URL` | ❌ | Override Gotify endpoint (default `http://127.0.0.1:18080`). |
| `GOTIFY_TOKEN` | ❌ | Pre-provisioned Gotify application token. If omitted, the app reads `cache/gotify_app_token.txt`. |
| `GOTIFY_DEFAULTUSER_NAME` | ❌ | Default Gotify admin username (embedded server). |
| `GOTIFY_DEFAULTUSER_PASS` | ❌ | Default Gotify admin password (change immediately).

### `caddy` service

| Variable | Required | Description |
| --- | --- | --- |
| `NOC_DOMAIN` | ✅ | Public hostname for the dashboard (`noc.example.com`). Use `localhost` for dev/local certs. |
| `GOTIFY_DOMAIN` | ❌ | Optional hostname for exposing embedded Gotify (`gotify.example.com`). |
| `ACME_EMAIL` | ✅ for public certs | Email address for Let’s Encrypt/ACME notifications. |

### Volumes

| Volume | Path | Contains |
| --- | --- | --- |
| `noc_cache` | `/var/www/html/cache` | Credentials, status cache, ping data, `metrics.sqlite`, Gotify data, logs. |
| `caddy_data` | `/data` | Let’s Encrypt certificates, keys, and account data. |
| `caddy_config` | `/config` | Caddy runtime configuration snapshots. |

To reset credentials, delete `cache/auth.json` inside `noc_cache`. To clear history metrics, remove `metrics.sqlite` while the container is stopped.

---

## Operations & Maintenance

* **Backups** – Snapshot the `noc_cache`, `caddy_data`, and `caddy_config` volumes. SQLite and Gotify DBs reside in `noc_cache`.
* **Logs** – App errors stream to container logs. Gotify delivery attempts append to `cache/gotify_log.txt`. TLS provisioning responses surface in the modal and container logs.
* **Upgrades** – Rebuild the image (`docker compose pull` or `docker compose build`). Persistent volumes retain credentials, tokens, and history.
* **Siren audio** – The browser loads `buz.mp3`. If autoplay is blocked, users can enable sound using the on-screen prompt.
* **Mobile access** – Use the Android wrapper or create a standalone PWA bookmark; the UI is responsive.

---

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| Can’t login | Delete `cache/auth.json` (in `noc_cache`) to restore `admin/admin`. |
| TLS issuance fails | Ensure DNS points to the host, ports 80/443 are reachable, and consider enabling staging in the TLS modal for testing. See [`docs/CADDY.md`](docs/CADDY.md). |
| No Gotify alerts | Confirm `GOTIFY_TOKEN`, review `cache/gotify_log.txt`, and try the in-app test notification. |
| History charts empty | Allow a few minutes for the metrics cron (per-minute inserts) to populate SQLite. |
| High latency badge | Inspect ping graph in the device modal; acknowledgements silence the siren but not Gotify latency alerts. |

---

## Development Notes

* **Tech stack:** PHP 8.2 + Apache, SQLite, Vanilla JS/HTML/CSS, Chart.js, Caddy 2, Gotify 2.6.
* **Local tooling:** `build-multiarch.sh` / `.ps1` handle cross-architecture builds. `docker-compose.yml` wires dependencies; comment out the Caddy service to run the app locally without TLS.
* **Coding conventions:** No framework, minimal dependencies. See `assets/app.js` for front-end logic and `device-detail.js` for modal rendering.
* **Android wrapper:** Open `android/` in Android Studio. Adjust the default URL in `MainActivity.kt` or pass `intent.putExtra("url", "https://noc.example.com")`.

---

## License

MIT License. See `LICENSE` (if present) or the project repository for details.
