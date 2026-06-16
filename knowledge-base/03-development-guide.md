# Development Guide

## Setup

```bash
go mod download
go build ./...
go test ./...
```

Node.js is needed for Playwright-based DOM checks.

## Common Workflows

### Add or change a protocol

1. Update the module in `protocol/`.
2. Add/adjust module tests.
3. Update frontend model/templates only if UI behavior changes.
4. Run `go test ./...` and Xray config validation.

### Change panel forms

1. Update FormSchema or template.
2. Update frontend model defaults/normalization if needed.
3. Rebuild binary to refresh embedded assets.
4. Run DOM checks.

### Change certificate behavior

1. Update `x-ui.sh` for command behavior.
2. Update `web/service/certificate.go` for API scan behavior.
3. Update UI selector if needed.
4. Verify no private key content is exposed.

## Validation Checklist

```bash
bash -n install.sh
bash -n x-ui.sh
go build ./...
go test ./...
scripts/rc_smoke.sh
scripts/rc_upgrade_smoke.sh
node scripts/share_link_validate.js
```
