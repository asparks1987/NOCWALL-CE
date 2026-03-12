# Connector Compatibility Matrix (Closed Beta v1)

This matrix defines the current connector surface for UISP, Cisco, Juniper, Meraki, and the vendor-agnostic Generic HTTP feed.

## Scope

- Objective: read-only inventory/status polling with connectivity health visibility.
- Current implemented named connectors: `UISP`, `Cisco`, `Juniper`, `Meraki`.
- Generic HTTP is available as an interoperability bridge for unsupported hardware that can expose device status as JSON.
- Additional connector families are still being added one at a time as vendor docs and test access become available.

## Matrix

| Connector | API Endpoints | Settings UI Onboarding | Config Vars | Auth Modes | Demo Mode | Status |
|---|---|---|---|---|---|---|
| UISP | `POST /sources/uisp/poll`, `GET /sources/uisp/status` | Yes (`Account Settings` -> `Add NMS Source`) | `UISP_URL`, `UISP_TOKEN`, `UISP_DEVICES_PATH`, `UISP_POLL_INTERVAL_SEC`, `UISP_POLL_RETRIES` | `X-Auth-Token` (+ bearer fallback sent by connector) | Yes (`demo=true` or missing creds) | Supported |
| Cisco v1 | `POST /sources/cisco/poll`, `GET /sources/cisco/status` | Yes (`Account Settings` -> `Add NMS Source`) | `CISCO_URL`, `CISCO_TOKEN`, `CISCO_DEVICES_PATH`, `CISCO_AUTH_SCHEME`, `CISCO_POLL_INTERVAL_SEC`, `CISCO_POLL_RETRIES` | `bearer`, `x-auth-token`, `token`, `authorization`, `none` | Yes (`demo=true` or missing creds) | Supported |
| Juniper v1 | `POST /sources/juniper/poll`, `GET /sources/juniper/status` | Yes (`Account Settings` -> `Add NMS Source`) | `JUNIPER_URL`, `JUNIPER_TOKEN`, `JUNIPER_DEVICES_PATH`, `JUNIPER_AUTH_SCHEME`, `JUNIPER_POLL_INTERVAL_SEC`, `JUNIPER_POLL_RETRIES` | `bearer`, `x-auth-token`, `token`, `authorization`, `none` | Yes (`demo=true` or missing creds) | Supported |
| Meraki v1 | `POST /sources/meraki/poll`, `GET /sources/meraki/status` | Yes (`Account Settings` -> `Add NMS Source`) | `MERAKI_URL`, `MERAKI_TOKEN`, `MERAKI_DEVICES_PATH`, `MERAKI_AUTH_SCHEME`, `MERAKI_POLL_INTERVAL_SEC`, `MERAKI_POLL_RETRIES` | `x-cisco-meraki-api-key` | Yes (`demo=true` or missing creds) | Supported |
| Generic HTTP | n/a (web account source feed consumed by `?ajax=devices`) | Yes (`Account Settings` -> `Add NMS Source`) | per-account source `url`, `api_path`, `auth_scheme`, `token` | `bearer`, `x-auth-token`, `token`, `authorization`, `none` | Yes (local mock JSON feed in smoke coverage) | Supported |

## Smoke Validation

Run API connector tests:

```bash
docker run --rm -v ${PWD}:/work -w /work/api golang:1.22 go test ./...
```

Run closed-beta web smoke coverage (includes settings UI onboarding and wallboard ingestion for UISP/Cisco/Juniper/Meraki/Generic HTTP):

```bash
BASE_URL=http://localhost/app bash scripts/closed-beta-v1-smoke.sh
```

Optional runtime checks (API running at `http://localhost:8080`):

```bash
curl -sS -X POST http://localhost:8080/sources/uisp/poll -H "Content-Type: application/json" -d '{"demo":true,"limit":25}'
curl -sS -X POST http://localhost:8080/sources/cisco/poll -H "Content-Type: application/json" -d '{"demo":true,"limit":25}'
curl -sS -X POST http://localhost:8080/sources/juniper/poll -H "Content-Type: application/json" -d '{"demo":true,"limit":25}'
```
