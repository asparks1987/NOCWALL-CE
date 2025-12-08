# Phase 4 Item 9 - API Contract Tests

## Scope
- Success and failure paths for auth, inventory, incidents, metrics, notifications, audit.
- Schema validation against OpenAPI; snapshot important responses.

## Tooling
- Pact/contract tests or supertest/jest (if TS); integration tests hitting ephemeral DB/Redis.

## Error/Diagnostics
- Test failures emit clear messages with request payloads and expected vs actual; include request_id when available.
