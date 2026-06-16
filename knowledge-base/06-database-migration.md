# Database Migration

Migrations live in `database/migration.go` and are tracked by `schema_migrations`.

## Current Migrations

- `password_hash`
- `inbound_clients`
- `vless_decryption_none`

## Rules

- Migrations must be idempotent.
- Back up DB before applying pending migrations.
- Use transactions.
- Do not rewrite already-applied migration meaning.
- Validate legacy DB behavior with `scripts/rc_upgrade_smoke.sh`.
