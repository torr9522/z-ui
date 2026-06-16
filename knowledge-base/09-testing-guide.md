# Testing Guide

## Required Local Commands

```bash
bash -n install.sh
bash -n x-ui.sh
go build ./...
go test ./...
```

## Smoke Tests

```bash
scripts/rc_smoke.sh
scripts/rc_upgrade_smoke.sh
```

## JavaScript Checks

```bash
node scripts/share_link_validate.js
```

DOM checks require a reachable panel and credentials:

```bash
XUI_BASE_URL=http://127.0.0.1:PORT XUI_USERNAME=... XUI_PASSWORD=... node scripts/ui_dom_check.js
```
