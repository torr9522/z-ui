# Protocol System

Protocol behavior is centered on `protocol.Module`.

## Interface

- `Name() string`
- `BuildInbound(*model.Inbound)`
- `Validate(*model.Inbound)`
- `Migrate(*model.Inbound)`
- `FormSchema() FormSchema`

## Registered Protocols

- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- SOCKS
- HTTP
- mixed
- tunnel

## Compatibility Rules

- VMess legacy `alterId` is normalized to `0`.
- VLESS always gets `decryption=none`.
- VLESS legacy origin/direct flow maps to Vision.
- VLESS splice is removed.
- Trojan flow is removed from modern paths.
- XHTTP writes `xhttpSettings`; legacy `httpSettings` and `splithttpSettings` are read-compatible.
- Non-stream protocols clear irrelevant stream/sniffing fields on modern saves.
