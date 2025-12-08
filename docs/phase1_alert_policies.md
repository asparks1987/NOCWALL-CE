# Phase 1 Item 6 - Alert Policy Defaults (Per Role)

Defaults to refine in later phases; include verbose diagnostics for banner/logs.

## Roles Covered
- Gateways
- Routers/switches
- Access points (including PTP/wireless backbone)

## Downtime Thresholds
- Initial alert after 30s offline (gateway/backbone/AP).
- Repeat every 10m while offline.
- Dependency-aware: suppress child (AP/switch) alerts if upstream gateway offline.

## Flap Detection
- Threshold: >=3 transitions in 15m.
- Suppress repeats for 30m after a flap alert.

## Latency/Jitter
- Sustained latency: >=200 ms for 3 consecutive samples (per device).
- Packet loss (if available): >5% over 5m window triggers warning; >10% triggers critical.
- Jitter (if available): >50 ms over 5m window triggers warning.

## Utilization/Performance
- CPU warn/crit: 75% / 90% sustained for 5m.
- RAM warn/crit: 75% / 90% sustained for 5m.
- Temperature warn/crit: device-specific if available; default 70C warn, 80C crit.
- Throughput/bandwidth (if exposed): configurable thresholds per site/role.

## Notifications
- Channels: Gotify, FCM push, email/SMS/webhook; prioritize push + Gotify for on-call.
- Repeat cadence respects suppression windows; escalation after N repeats to secondary/on-call lead.

## Maintenance & Quiet Hours
- Per-site/device/role maintenance windows mute alerts; post-window validation pings.
- Quiet hours per user/team; allow override for critical categories (gateway down).

## Banner/Diagnostics Requirements
- Include alert type, device/site, role, thresholds breached, samples, and correlation ID.
- For suppressed alerts (dependency/maintenance), log reason and show badge in UI “suppressed (gateway offline)” with details.
