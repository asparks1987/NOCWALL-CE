# Phase 9 Item 2 - API Client & Realtime

## Features
- Retrofit client for REST; WebSocket/SSE client for live updates.
- Automatic token refresh; retry with backoff.

## Error/Diagnostics
- Surface request_id on failures; banner shows connectivity status (API/WS/push) and last error.
