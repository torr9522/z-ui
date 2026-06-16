# Install Script

## install.sh

Performs secure unattended install:

- root check
- OS/arch detection
- dependency install
- release package download
- install to `/usr/local/x-ui`
- install command to `/usr/bin/x-ui`
- install systemd service
- random username/password/port
- service start and verification

## x-ui.sh

Operational command:

- service control
- reset user
- reset port
- info
- certificate manager
- update
- uninstall with explicit confirmation

Do not modify update or destructive behavior without explicit validation and rollback considerations.
