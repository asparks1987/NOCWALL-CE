# Phase 3 Item 7 - Staging Environment & Test Data

## Staging Setup
- Mirrors prod topology with smaller scale; uses staging domains and ACME staging certs.
- Uses test OIDC client and FCM project.
- Seed UISP-like data or connect to test UISP controller.

## Test Data
- Seed sites/devices/incidents/metrics for dashboards and alerts.
- Seed users with roles (admin/operator/viewer/wallboard) and sample notification endpoints.

## Observability
- Enable verbose logging and banners; synthetic monitors for poller/notifications.

## Error/Diagnostics
- Expose staging health/dashboard to devs; banner shows staging-specific issues with request_id.
