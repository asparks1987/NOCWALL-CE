# Phase 1 Item 7 - Notification Channels & Escalation Rules

## Channels (initial)
- Gotify (existing, continues).
- FCM push (Android + browser service worker).
- Email.
- SMS/webhook (Teams/Slack/Generic HTTP).
- Optional voice/phone escalation (future).

## Delivery & Fan-out
- Notification router fans out per user preference and severity; batches duplicate alerts.
- Actionable notifications support Ack/Clear/Runbook link where supported (push/webhook).
- Dead-letter queue for failed deliveries with retries and backoff.

## Escalation Rules (baseline)
- Initial send on trigger (per alert policies).
- Repeat every 10m (offline) or per policy until acknowledged or recovered.
- Escalate to secondary/on-call lead after 2 repeats or configurable threshold.
- Quiet hours respected unless critical (gateway down).

## Registration & Tokens
- Device/browser tokens stored per environment; rotate on logout; expire stale tokens.
- Web push uses service worker; Android uses FCM; both include correlation/request IDs for debugging.

## Error/Banner Expectations
- Delivery errors surfaced with channel, code, last attempt time, and correlation ID in UI banner/logs.
- Per-user debug view showing last notification attempts and statuses.
