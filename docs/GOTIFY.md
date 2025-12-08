# Embedded Gotify

The UISP NOC container bundles a Gotify 2.6 server for push notifications. Gotify starts automatically alongside Apache, stores its data inside the shared `cache/` volume, and is used to deliver offline/online, flapping, and latency alerts. The Android companion app consumes the same stream so sirens/vibration fire on handhelds the instant an outage lands.

---

## Roadmap Note

Gotify remains supported, but the upcoming notification service will fan out to multiple channels (Gotify, FCM push, email/SMS/webhooks) with throttling and actionable notifications. See `docs/PROJECT_PLAN.md` (Phases 6–7) for how Gotify fits into the multi-channel plan.

---

## How It Works

* `start.sh` launches Gotify before Apache with configuration at `/etc/gotify/config.yml` (listening on `0.0.0.0:18080`).
* When the container boots, the app looks for `cache/gotify_app_token.txt`. If it does not exist, a new application token is generated and written to that file for the UI to consume.
* Alerts are sent via Gotify’s HTTP API (`/message`). Delivery attempts and failures are appended to `cache/gotify_log.txt` for auditing.
* The dashboard exposes a **Send Test Notification** button (Settings menu) that exercises the currently configured token.

---

## Default Credentials (embedded server)

| Variable | Default | Notes |
| --- | --- | --- |
| `GOTIFY_DEFAULTUSER_NAME` | `admin` | Admin user created on first boot. Change after logging in. |
| `GOTIFY_DEFAULTUSER_PASS` | `changeme` | Update immediately to secure the Gotify UI/API. |

Change these values in `docker-compose.yml` or set them as environment overrides before the first run. Credentials are stored under `cache/gotify/`.

---

## Using the Embedded Server

1. Start the stack (`docker compose up -d`).
2. Optionally access the Gotify UI:
   * From inside the Docker network: `http://uisp-noc:18080`
   * From the host (if ports are exposed): `http://localhost:18080`
   * Through Caddy (if `GOTIFY_DOMAIN` is defined): `https://<GOTIFY_DOMAIN>/`
3. Log in with the default credentials and create applications if you prefer to manage tokens manually.
4. Check `cache/gotify_app_token.txt` for the auto-provisioned token that the dashboard already uses.

---

## Pointing to an External Gotify Instance

Set the following `uisp-noc` environment variables:

```env
GOTIFY_URL=https://gotify.example.com
GOTIFY_TOKEN=<your application token>
```

Restart the container. The embedded Gotify server will still start (for completeness) but will not be used for notifications.

---

## Android Companion Hooks

* Bundle the auto-generated application token (from `cache/gotify_app_token.txt`) into your Android build config if you want the wrapper to preflight notifications.
* The WebView mirrors audio/vibration from Gotify messages. Long-press the siren toggle in the dashboard if you want to keep the phone buzzing during maintenance drills.
* When publishing the companion app via an MDM, keep Gotify reachable over HTTPS (either via the bundled Caddy host or your own proxy) so background alerts survive when the device screen is off.

---

## Notifications Sent

| Trigger | Details |
| --- | --- |
| Gateway/router/switch offline | Fires after ~30 seconds of downtime and repeats every 10 minutes until recovery. |
| Gateway/router/switch online | Sent immediately when service returns. Clears offline notification timers. |
| Flapping | Raised when a backbone device flaps ≥3 times within 15 minutes (suppressed for 30 minutes after each alert). |
| Sustained latency | Fired when ping latency is ≥200 ms for 3 consecutive samples (suppressed for 15 minutes between sends). |

Acknowledging a device in the UI silences the siren but does **not** block Gotify notifications.

---

## Maintenance & Logs

* Embedded Gotify data lives under `cache/gotify/` in the `noc_cache` volume; back it up with the rest of the app state.
* Review `cache/gotify_log.txt` for delivery errors or missing tokens.
* To rotate the token, delete `cache/gotify_app_token.txt` and restart. A new token will be generated automatically (update any external consumers accordingly).

---

## Hardening Tips

* Change the default admin password immediately and create scoped application tokens per integration.
* Restrict public exposure by omitting `GOTIFY_DOMAIN` unless you need the UI remotely.
* If publishing Gotify externally, add authentication/authorization in front of it (e.g., Authentik, Authelia, OAuth proxy).
