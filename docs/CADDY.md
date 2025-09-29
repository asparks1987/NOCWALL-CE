TLS + Caddy + Simple Sign-On

Overview
- Caddy terminates TLS and reverse-proxies to the app and the embedded Gotify. Simple sign-on is implemented inside the app (default admin/admin), with a change-password option after login.

Files
- `Caddyfile`: Caddy configuration at repo root.
- `docker-compose.yml`: Includes a `caddy` service wired to the app.

Environment
- `NOC_DOMAIN`: Public hostname for UISP NOC (e.g., `noc.example.com`). Defaults to `localhost` if unset.
- `GOTIFY_DOMAIN` (optional): Hostname for embedded Gotify (e.g., `gotify.example.com`). Omit to keep Gotify internal.
- `ACME_EMAIL`: Email for Let's Encrypt/ACME.

App Sign-On
- Default username: `admin`
- Default password: `admin`
- After logging in, click "Change Password" in the header to set a new password. Credentials persist in `cache/auth.json`.

Bring Up
```bash
docker compose up -d
```

Access
- UISP NOC: `https://<NOC_DOMAIN>/` (app login required)
- Gotify (optional): `https://<GOTIFY_DOMAIN>/` (no app login; consider leaving this internal)

UI Certificate Provisioning
- Open the app and login → click "TLS/Certs".
- Enter Domain and ACME email (and optional Gotify domain), then Provision.
- This uses Caddy's admin API to load a live config and begin ACME issuance.
- Option: Use Let's Encrypt Staging for dry runs to avoid rate limits.

Notes
- Without public DNS: set `NOC_DOMAIN=localhost`. Caddy uses local certs (you may need to trust Caddy's local CA).
- If you prefer not to expose Gotify publicly, don't set `GOTIFY_DOMAIN`.
- You can bypass Caddy and map ports directly on `uisp-noc` (see commented `ports`).
