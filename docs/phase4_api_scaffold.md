# Phase 4 Item 1 - API Scaffold (Auth & RBAC)

## Stack
- TypeScript + NestJS/Express (or Go alternative) with OIDC client, JWT validation, session cookies.

## Auth/RBAC
- Middlewares for access token validation, refresh flow, PAT auth.
- RBAC guard mapping roles (admin/operator/viewer/wallboard) to routes.
- CSRF protection for cookie-based sessions; rate limits on auth endpoints.

## Error/Banner Expectations
- Standard error shape `{ code, message, request_id, details }`.
- Verbose logs on auth failures; banner shows reason and request_id.
