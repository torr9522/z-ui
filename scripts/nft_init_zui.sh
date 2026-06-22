#!/usr/bin/env bash
set -euo pipefail

TABLE_FAMILY="${ZUI_NFT_TABLE_FAMILY:-inet}"
TABLE_NAME="${ZUI_NFT_TABLE_NAME:-zui_port_guard}"
DRY_RUN="${ZUI_NFT_DRY_RUN:-0}"

run_nft() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf 'nft %s\n' "$*"
    return 0
  fi
  nft "$@"
}

ensure_nft() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    return 0
  fi
  command -v nft >/dev/null 2>&1 || {
    printf 'nftables 不可用: 未找到 nft 命令\n' >&2
    exit 1
  }
}

table_exists() {
  [[ "${DRY_RUN}" == "1" ]] && return 1
  nft list table "${TABLE_FAMILY}" "${TABLE_NAME}" >/dev/null 2>&1
}

chain_exists() {
  local chain="$1"
  [[ "${DRY_RUN}" == "1" ]] && return 1
  nft list chain "${TABLE_FAMILY}" "${TABLE_NAME}" "${chain}" >/dev/null 2>&1
}

set_exists() {
  local set_name="$1"
  [[ "${DRY_RUN}" == "1" ]] && return 1
  nft list set "${TABLE_FAMILY}" "${TABLE_NAME}" "${set_name}" >/dev/null 2>&1
}

ensure_nft

if ! table_exists; then
  run_nft add table "${TABLE_FAMILY}" "${TABLE_NAME}"
fi

if ! chain_exists input; then
  run_nft add chain "${TABLE_FAMILY}" "${TABLE_NAME}" input '{ type filter hook input priority 0; policy accept; }'
fi

if ! chain_exists output; then
  run_nft add chain "${TABLE_FAMILY}" "${TABLE_NAME}" output '{ type filter hook output priority 0; policy accept; }'
fi

if ! set_exists blocked_ports; then
  run_nft add set "${TABLE_FAMILY}" "${TABLE_NAME}" blocked_ports '{ type inet_service; flags timeout,dynamic; }'
fi

printf 'z-ui nftables bootstrap 已完成: table %s %s\n' "${TABLE_FAMILY}" "${TABLE_NAME}"
