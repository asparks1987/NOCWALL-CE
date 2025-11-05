# UISP NOC – Container Summary

**UISP NOC** is a self-hosted network operations dashboard for Ubiquiti UISP. It polls UISP every few seconds, shows gateways/routers/CPEs with health badges, stores short-term history in SQLite, sounds an audible siren, and ships with an embedded Gotify server for offline/online, flapping, and latency notifications.

## Highlights

- One container: PHP 8.2 + Apache UI, SQLite metrics, siren audio, TLS modal, and embedded Gotify.
- Caddy sidecar (included in `docker-compose.yml`) terminates TLS and can publish Gotify on a second hostname.
- Acknowledgements (30m → 12h), outage simulation, history charts, and per-device latency/flap alerts.
- Simple sign-on (default `admin/admin`, persisted in volume) with password change dialog.

## Quick Start

```bash
docker compose up -d
```

Set these minimum environment variables before launching:

- `UISP_URL` – e.g., `https://your-uisp.unmsapp.com`
- `UISP_TOKEN` – UISP API token
- `NOC_DOMAIN` (Caddy) – public hostname for HTTPS
- Optional: `SHOW_TLS_UI=1` to enable in-app Let’s Encrypt provisioning

Volumes persist credentials, Gotify data, and metrics (`noc_cache`, `caddy_data`, `caddy_config`).

## Notifications

Gotify auto-generates an application token at `cache/gotify_app_token.txt`. Use the dashboard’s **Send Test Notification** button to verify delivery or point to an external Gotify by setting `GOTIFY_URL` + `GOTIFY_TOKEN`.

## Need More Details?

Full documentation, TLS walkthroughs, and Android wrapper instructions live at https://github.com/UISP-NOC/UISP-NOC.
