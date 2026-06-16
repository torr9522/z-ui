# Certificate System

Certificates are stored under `/etc/x-ui/certs/<name>/`.

Expected files:

- `fullchain.pem`
- `privkey.pem`
- `meta.json`

## Features

- Import certificate
- Delete certificate with backup
- Set panel HTTPS certificate
- Issue domain certificate with acme.sh standalone HTTP-01
- Experimental IP certificate entry
- Renewal timer and status
- TLS selector in inbound forms

## Safety

- API returns metadata and file paths only.
- API does not return private key contents.
- Imported certificates cannot enable ACME auto-renew.
- Backup/deleted/test cert directories are filtered from lists.
