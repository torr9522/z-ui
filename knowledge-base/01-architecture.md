# Architecture

## Main Flow

UI -> Controller -> Service -> ProtocolModule -> Runtime Builder -> Runtime Manager -> Xray

## Database

SQLite stores users, settings, inbounds, inbound clients, and schema migrations. The DB is the source of truth. Runtime state is rebuilt or reconciled from DB state.

## Web Layer

`web/web.go` initializes Gin, sessions, CSRF, embedded templates, embedded assets, and route registration.

## Controllers

`web/controller/` binds routes to services. Controllers should stay thin: bind input, enforce login/CSRF, call service, return JSON/template.

## Services

`web/service/` contains business logic. Important services:

- `InboundService`: validates, migrates, saves, and loads inbounds/clients.
- `XrayService`: builds runtime snapshots and reconciles Xray.
- `SettingService`: panel and Xray settings.
- `CertificateService`: scans certificate metadata.

## ProtocolModule

`protocol.Module` owns per-protocol behavior:

- `BuildInbound`
- `Validate`
- `Migrate`
- `FormSchema`

Do not add protocol-specific generation in `XrayService`.

## Runtime

`runtime/` compares desired snapshots with current materialized state. It uses Runtime API where safe and falls back to full restart when needed.

## Install and Operations

`install.sh` bootstraps a server. `x-ui.sh` is the operational command for service, credential, port, certificate, renewal, and uninstall actions.
