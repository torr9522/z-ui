# Project Overview

Modern x-ui is a maintained x-ui modernization branch for Xray-core 26.x. The project keeps the original lightweight Go + SQLite + server-rendered panel shape, while adding safer authentication, protocol modules, Runtime API reconciliation, migrations, certificate management, and modern Xray protocol support.

## Goals

- Keep the project maintainable by one developer.
- Keep SQLite and settings JSON compatibility where practical.
- Make the database the desired state and Xray runtime the materialized state.
- Centralize protocol behavior through ProtocolModule.
- Preserve legacy reads while modernizing new saves.

## Current Baseline

- Go: 1.26
- Xray-core: 26.6.1
- DB: SQLite
- Panel: existing Vue/Ant Design templates and embedded assets
- Service manager: systemd

## Supported Protocols

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

## Core Modern Features

- Runtime API with full restart fallback
- ProtocolModule config generation
- `schema_migrations`
- `password_hash`
- `inbound_clients`
- REALITY, Vision, XHTTP, mixed
- HTTP + TLS mode
- Certificate Manager and ACME lifecycle
- Secure install and management scripts
