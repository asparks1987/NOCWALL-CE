# TLS, Caddy, and Reverse Proxying

Caddy ships alongside the UISP NOC container to terminate TLS, publish the dashboard on ports 80/443, and optionally expose the embedded Gotify instance on its own hostname. The application can also instruct Caddy to request Let's Encrypt certificates via the admin API when the optional TLS UI is enabled, which keeps the Android companion app from throwing certificate warnings in its locked-down WebView.

---

## Roadmap Note

As part of the platform revamp (see `docs/PROJECT_PLAN.md`), Caddy remains the TLS terminator and reverse proxy but will front multiple services: API, SPA web app, notification endpoints, and optional Gotify. Expect updated Compose/Helm wiring once the new services land.

---

## Files of Interest

| File | Purpose |
| --- | --- |
| `Caddyfile` | Baseline reverse-proxy configuration loaded at container start. Routes traffic for the dashboard and (optionally) Gotify. |
| `docker-compose.yml` | Declares the `caddy` service, volumes, and environment variables. |
| `index.php` | Hosts the TLS provisioning modal logic and calls the Caddy admin API when the UI workflow is used. |

---

## Environment Variables (`caddy` service)

| Variable | Description |
| --- | --- |
| `NOC_DOMAIN` | Primary hostname for the dashboard. Defaults to `localhost`, which causes Caddy to issue local development certificates. |
| `GOTIFY_DOMAIN` | Optional hostname for exposing the embedded Gotify server. Leave unset to keep Gotify internal-only. |
| `ACME_EMAIL` | Email address passed to Let’s Encrypt/ACME. Required for public certificates. |

---

## Launching Caddy

1. Set `NOC_DOMAIN` (and optionally `GOTIFY_DOMAIN`) in `docker-compose.yml`.
2. Provide an ACME email address.
3. Start the stack: `docker compose up -d`.
4. Browse to `https://<NOC_DOMAIN>/` once certificates are issued.

Caddy writes certificates and ACME account data into the `caddy_data` volume and caches runtime configuration in `caddy_config`.

---

## Using the In-App TLS Provisioning UI

The TLS modal is hidden by default. Enable it by setting `SHOW_TLS_UI=1` on the `uisp-noc` service and restarting. Once visible:

1. Open the dashboard and click **TLS / Certs**.
2. Review the current live Caddy configuration fetched from `http://caddy:2019/config/`.
3. Provide:
   * **Domain** – Hostname for the dashboard (must resolve publicly to your host).
   * **Gotify Domain** (optional) – Separate hostname for the embedded Gotify UI/API.
   * **ACME Email** – Used for Let’s Encrypt notifications.
   * **Use Staging** – Toggle when testing to avoid rate limits.
4. Submit. The app loads the configuration into Caddy via `http://caddy:2019/load`. On success, Caddy immediately begins HTTP-01 validation and certificate issuance.

### Requirements

* Public DNS A/AAAA records must point to your server.
* Ports 80 and 443 must be open to the internet.
* For staging/localhost tests, Caddy can serve locally-trusted certificates when `NOC_DOMAIN=localhost`.

### Troubleshooting UI Provisioning

| Symptom | Resolution |
| --- | --- |
| Modal shows `caddy_unreachable` | Verify the `caddy` container is running and reachable on the internal Docker network. |
| Issuance fails with validation errors | Confirm public DNS is correct, ports 80/443 are forwarded, and ACME staging is disabled for production. Review the Caddy container logs for detailed errors. |
| Need to revert to the static `Caddyfile` | Restart the Caddy container to reload the baked-in `Caddyfile`. |

---

## Manual Configuration

You can skip the UI workflow and rely solely on the static `Caddyfile`:

* Map additional hostnames using `handle`/`reverse_proxy` blocks.
* Serve the dashboard without TLS by running behind another reverse proxy and omitting the certificate automation directives.
* To expose Gotify, uncomment the relevant block in `Caddyfile` and set `GOTIFY_DOMAIN`.

After editing `Caddyfile`, restart the container (`docker compose restart caddy`).

---

## Android Companion Considerations

* Use valid HTTPS names (`NOC_DOMAIN` and optional `GOTIFY_DOMAIN`) that match what you hard-code into the Android companion app to avoid WebView SSL prompts.
* If you plan to deep-link from Gotify push notifications into the app, expose Gotify over TLS via Caddy so Android treats the endpoint as trusted while the device is asleep.
* Ship updated certificates (or pin your public hostname) before distributing a new APK through your MDM so techs never see certificate rotation prompts during outages.

---

## Bypassing Caddy

For lab or air-gapped environments, expose the app directly by uncommenting the `ports` section on the `uisp-noc` service (`1200:80`, `18080:18080`) and stopping/removing the Caddy service. In this mode TLS is not provided; terminate TLS with another proxy or use HTTP.

