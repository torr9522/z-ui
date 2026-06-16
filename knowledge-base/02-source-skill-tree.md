# Source Skill Tree

This file is the fastest entry point for future development.

## Project Directory Tree

- `config/`: embedded project name/version and asset version helpers.
- `database/`: SQLite initialization, migrations, and DB tests.
- `database/model/`: GORM model definitions.
- `logger/`: project logging wrapper.
- `protocol/`: ProtocolModule implementations and protocol tests.
- `runtime/`: Runtime API client, planner, builder, manager, reconciler.
- `scripts/`: smoke tests, DOM checks, share-link validation, install smoke.
- `util/`: common helpers, JSON helpers, password hashing, random generation, sys helpers.
- `v2ui/`: legacy migration support from v2-ui.
- `web/`: panel server, controllers, services, templates, static JS/CSS assets.
- `web/assets/`: frontend model and vendor assets embedded in release mode.
- `web/html/`: server-rendered templates and form fragments.
- `xray/`: Xray process/config wrapper.
- `install.sh`: secure unattended install script.
- `x-ui.sh`: management command script.

## Core Files

- `main.go`: CLI entrypoint, run/setting/version commands.
- `config/config.go`: app version and asset cache-busting version.
- `database/db.go`: DB open/init and first-run credential bootstrap.
- `database/migration.go`: schema migration ledger and upgrade steps.
- `protocol/interface.go`: ProtocolModule contract.
- `protocol/registry.go`: protocol registration list.
- `runtime/builder.go`: DB inbound -> Xray raw/core snapshot.
- `runtime/manager.go`: Runtime API sync and restart fallback.
- `runtime/planner.go`: runtime diff planning.
- `runtime/client.go`: Xray Runtime API operations.
- `web/service/inbound.go`: inbound persistence, client split/attach, module validation.
- `web/service/xray.go`: Xray snapshot/reconcile service.
- `web/service/certificate.go`: local certificate scanner API logic.
- `web/controller/protocol.go`: protocol schema and REALITY keypair API.
- `web/controller/certificate.go`: certificate scan API.
- `web/html/xui/form/`: inbound form templates.
- `web/assets/js/model/xray.js`: frontend inbound model, normalization, share links.

## Modify Protocols

Change these first:

1. `protocol/<protocol>.go`
2. `protocol/registry.go` if adding/removing module registration
3. `protocol/*_test.go`
4. `web/assets/js/model/xray.js` only for frontend model/share links
5. `web/html/xui/form/protocol/*` only for protocol-specific templates

Do not put protocol-specific generation into `web/service/xray.go` or `runtime/builder.go`.

## Modify FormSchema

- Backend schema: `protocol/<protocol>.go`
- API: `web/controller/protocol.go`
- Renderer: `web/html/xui/form/protocol/schema.html`
- Frontend model defaults: `web/assets/js/model/xray.js`
- DOM validation: `scripts/ui_dom_check.js`

## Modify UI

- Templates: `web/html/`
- JS model/behavior: `web/assets/js/`
- Keep existing Vue/Ant Design stack.
- Release mode uses embedded assets; rebuild binary after frontend changes.

## Modify Certificates

- Shell command behavior: `x-ui.sh`
- API scanner: `web/service/certificate.go`
- API controller: `web/controller/certificate.go`
- TLS selector UI: `web/html/xui/form/tls_settings.html`
- DOM test: `scripts/certificate_ui_check.js`

Certificates are for TLS and panel HTTPS. Do not mix certificate logic into REALITY.

## Modify Install Script

- Bootstrap install only: `install.sh`
- Operations after install: `x-ui.sh`
- Install validation: `scripts/install_smoke.sh`

Do not silently change update/uninstall behavior without explicit safety checks.

## Modify Runtime

- API calls: `runtime/client.go`
- Diff planning: `runtime/planner.go`
- State sync/restart fallback: `runtime/manager.go`
- Snapshot building: `runtime/builder.go`
- Recovery orchestration: `runtime/reconciler.go`

DB remains desired state. Runtime remains materialized state.

## Modify Database

- Models: `database/model/model.go`
- Migrations: `database/migration.go`
- DB bootstrap: `database/db.go`
- Upgrade validation: `scripts/rc_upgrade_smoke.sh`

All migrations must be idempotent and backed by tests/smoke coverage.

## Add Tests

- Protocol logic: `protocol/*_test.go`
- DB/migrations: `database/*_test.go`
- Web service logic: `web/service/*_test.go`
- Share links: `scripts/share_link_validate.js`
- DOM: `scripts/ui_dom_check.js`, `scripts/certificate_ui_check.js`
- Install: `scripts/install_smoke.sh`

## Do Not Casually Change

- `protocol/interface.go`: core protocol contract.
- `runtime/manager.go`: state safety and fallback behavior.
- `database/migration.go`: already-applied migration semantics.
- `web/assets` vendor files.
- `install.sh` and `x-ui.sh` destructive commands.
- Any code path that writes credentials, private keys, or DB files.
