#!/usr/bin/env bash
set -euo pipefail

config_path="${1:-packaging/logrotate/x-ui-xray-access}"

grep -Eq '^/var/log/xray/access\.log[[:space:]]+\{$' "${config_path}"
for directive in daily "rotate 7" compress delaycompress missingok notifempty copytruncate; do
  grep -Eq "^[[:space:]]*${directive}[[:space:]]*$" "${config_path}"
done
if grep -Eq 'cron|rm[[:space:]]+-|nftables|iptables|firewall|conntrack|Port Guard|port-guard|ban|drop|block' "${config_path}"; then
  echo "logrotate config contains forbidden control or delete logic" >&2
  exit 1
fi
