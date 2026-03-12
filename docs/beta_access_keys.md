# Closed Beta Access Keys Runbook

This runbook covers issuing, redeeming, rotating, and revoking closed beta keys that grant temporary PRO access.

## Storage Model

- File: `cache/users.json`
- Top-level:
  - `beta_keys` map keyed by `code`
  - `beta_key_audit` event array
- Per-user:
  - `users.<username>.beta_access`

Key schema fields:

- `code`
- `status` (`active`, `disabled`, `expired`)
- `max_redemptions`
- `redeemed_count`
- `expires_at`
- `issued_by`
- `notes`
- `created_at`
- `updated_at`
- `redemptions` (`username -> redeemed_at`)

## Entitlement Order

1. Paid/trial subscription entitlement
2. Valid redeemed beta key entitlement
3. CE free entitlement

This ensures paid subscription remains highest priority.

## Issue a Key

Generate a new key (single-use, no expiry):

```bash
php scripts/beta_keys.php generate --issued-by ops
```

Generate with custom limits and expiry:

```bash
php scripts/beta_keys.php generate \
  --max-redemptions 25 \
  --ttl-days 30 \
  --issued-by ops \
  --notes "Closed beta wave 1"
```

Generate a specific code:

```bash
php scripts/beta_keys.php generate --code NOCWALL-BETA-TEAM1-0001 --issued-by ops
```

## Redeem a Key

- At signup: optional `Closed beta key` field on the register form.
- Existing account: `Account Settings -> Redeem Beta Key`.

Successful redemption writes:

- `beta_keys.<code>.redemptions.<username>`
- `users.<username>.beta_access`
- `beta_key_audit` (`beta_key_redeem`)

## List Keys

```bash
php scripts/beta_keys.php list
```

JSON output:

```bash
php scripts/beta_keys.php list --json
```

## Revoke / Disable a Key

Disable immediately (revokes beta entitlement for linked accounts):

```bash
php scripts/beta_keys.php disable --code NOCWALL-BETA-TEAM1-0001 --reason "abuse report"
```

Alias:

```bash
php scripts/beta_keys.php revoke --code NOCWALL-BETA-TEAM1-0001
```

Re-enable:

```bash
php scripts/beta_keys.php enable --code NOCWALL-BETA-TEAM1-0001
```

Revocation/enable operations append audit events (`beta_key_revoke`, `beta_key_enable`) and update linked user beta state.

## Expiration Handling

- Keys with `expires_at` in the past are treated as expired.
- Expired keys no longer grant entitlement.
- Expiration can be set at issuance with `--expires-at` or `--ttl-days`.
