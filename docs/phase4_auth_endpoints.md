# Phase 4 Item 2 - Users/Tokens/Session Endpoints

## Endpoints
- `POST /auth/login` (OIDC start), `POST /auth/refresh`, `POST /auth/logout`.
- `GET /me` for profile/roles.
- `POST /tokens` (create PAT/service token), `GET /tokens`, `DELETE /tokens/{id}`.

## Behavior
- Cookies HTTP-only for browser; Bearer for API clients.
- Tokens scoped with expiry; revoke tokens on logout/rotation.

## Error/Banner Expectations
- Errors include code (`auth_invalid`, `auth_expired`, `auth_csrf`, `token_scope_invalid`), message, request_id.
- UI banner shows cause and action (re-login, check scope).
