#!/usr/bin/env bash
set -euo pipefail

output="$(ZUI_NFT_DRY_RUN=1 scripts/nft_bootstrap.sh)"

grep -Fq 'systemctl enable nftables' <<<"${output}"
grep -Fq 'systemctl start nftables' <<<"${output}"
grep -Fq 'nft add table inet zui_port_guard' <<<"${output}"
grep -Fq 'nft add chain inet zui_port_guard input { type filter hook input priority 0; policy accept; }' <<<"${output}"
grep -Fq 'nft add chain inet zui_port_guard output { type filter hook output priority 0; policy accept; }' <<<"${output}"
grep -Fq 'nft add set inet zui_port_guard blocked_ports { type inet_service; flags timeout,dynamic; }' <<<"${output}"

if grep -Eq '\b(delete|flush|drop|reject|dnat|snat|masquerade)\b|iptables|ufw|firewall-cmd|conntrack' <<<"${output}"; then
  printf 'nft bootstrap dry-run contains forbidden operation\n' >&2
  printf '%s\n' "${output}" >&2
  exit 1
fi
