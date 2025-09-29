Embedded Gotify
================

Overview
- A Gotify server is bundled in the container and starts on port `18080`.
- Data (database) is stored under the existing `cache` volume so it persists across updates.

Quick Start
- Build and run the stack as usual. The Gotify UI becomes available at `http://<host-ip>:18080`.
- Log in with defaults from `docker-compose.yml`:
  - `GOTIFY_DEFAULTUSER_NAME=admin`
  - `GOTIFY_DEFAULTUSER_PASS=changeme`
- Create an Application named "UISP NOC" (or any name) and copy its token.
- Provide the token to the app via either:
  - Environment var: set `GOTIFY_TOKEN=<your token>` in `docker-compose.yml`, or
  - File: save the token in `./cache/gotify_app_token.txt` (container path `/var/www/html/cache/gotify_app_token.txt`).

External Gotify (optional)
- To use an external server, set:
  - `GOTIFY_URL=https://your-gotify.example.com`
  - `GOTIFY_TOKEN=<app token>`

What Is Notified
- A gateway going OFFLINE (after ~30s grace) and when it comes back ONLINE.
- Acknowledgements suppress the repeating siren in the UI; offline/online notifications are still sent on state changes.

Notes
- Change the default Gotify credentials on first login.
- The embedded server is intentionally minimal; for HA/backup scenarios, consider an external Gotify.

