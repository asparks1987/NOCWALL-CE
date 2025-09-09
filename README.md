# 📡 UISP NOC Dashboard

A self-hosted **Network Operations Center (NOC) dashboard** for monitoring **Ubiquiti UISP devices** (gateways + CPEs).  
Runs as a **Dockerized PHP + Apache app** with SQLite logging, live alerts, and a wallboard-style UI.

---

## ✨ Features

- 🔄 **Live Polling** — queries UISP API every few seconds (via AJAX).
- 🟢🔴 **Online / Offline Indicators** — green/red dots + flashing red cards for offline.
- 📊 **Device Metrics**:
  - Ping latency (color-coded)
  - CPU %, RAM %, temperature
  - Uptime
- 🚨 **Alerting**:
  - Loud siren (`buz.mp3`) if a gateway is offline ≥30s (unacked).
  - Acknowledgement system to silence alerts for a configurable period.
  - Outage simulation button for testing.
- 🗂 **Gateways & CPEs**:
  - Separate summaries with health percentages.
  - Offline devices pinned to the top.
- 🕒 **Historical Metrics (SQLite)**:
  - Logs CPU, RAM, Temp, Latency, Online state for all devices.
  - Keeps 7 days of history (older rows auto-pruned).
  - “View History” button → opens modal with **stacked Chart.js graphs**.
- 🐳 **Containerized Deployment**:
  - Runs on Raspberry Pi (ARMv7/ARM64) or x86 servers.
  - Environment-based config (UISP URL + API token).
  - Easy to launch via Portainer stack.

---

## 📦 Requirements

- Docker + Docker Compose (or Portainer)
- UISP instance with API token
- Gotify server (optional, for push notifications)

---

## ⚙️ Setup

### 1. Clone Repo
```bash
git clone https://github.com/asparks1987/UISP-NOC.git
cd uisp-noc
```

### 2. Add Your UISP Config
In `docker-compose.yml`, update:
```yaml
environment:
  - UISP_URL=https://your-uisp.unmsapp.com
  - UISP_TOKEN=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### 3. Build & Run
```bash
docker compose up --build -d
```

App will be available at:

```
http://<host-ip>:12443
```

---

## 🗂 Project Structure

```
uisp-noc/
├── Dockerfile
├── docker-compose.yml
├── index.php        # Main dashboard app
├── buz.mp3          # Siren sound for outages
└── cache/           # Holds status cache + SQLite DB
```

---

## 🔧 Development Notes

- **Cache & DB** are mounted at `./cache` → persisted across redeploys.
- SQLite file: `cache/metrics.sqlite`.
- Rotate old metrics automatically (7 days).
- For faster refresh in lab environments, lower AJAX poll interval in `index.php`.

---

## 🚀 Multi-Arch Docker Build

Build once for ARM + x86 and push to Docker Hub:

```bash
./build-multiarch.sh youruser/uisp-noc:latest
```

Then deploy in Portainer with:
```yaml
image: youruser/uisp-noc:latest
```

---

## 🔒 Security

- API token passed via Docker env vars (never hardcoded).
- Use Docker secrets or Portainer environment variables in production.
- Recommended: run behind a reverse proxy (Traefik, Nginx) with HTTPS.

---

## 📊 Roadmap

- [ ] Role-based access (basic auth / login screen)  
- [ ] Customizable thresholds for alerts  
- [ ] Export historical data (CSV/JSON)  
- [ ] Optional Grafana integration for long-term trending  

---

## 📝 License

MIT — free to use, modify, and share.  
