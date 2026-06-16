# Lessons Learned

- Release mode serves embedded assets; rebuild after frontend changes.
- Long-lived browser cache requires asset cache busting.
- ProtocolModule must own config generation, validation, migration, and form schema.
- Legacy compatibility should be read/normalize side; new saves should write modern config.
- Trojan does not use flow.
- VMess should not expose non-zero alterId.
- VLESS requires `decryption=none`.
- HTTP+TLS is not a new protocol; it is HTTP with TLS security.
- Certificates are separate from REALITY.
- Imported certificates cannot be ACME-renewed.
- Install/update/uninstall scripts are production-risk zones.
