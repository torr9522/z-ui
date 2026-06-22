#!/usr/bin/env bash
set -euo pipefail

DRY_RUN="${ZUI_NFT_DRY_RUN:-0}"

run_cmd() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '%s\n' "$*"
    return 0
  fi
  "$@"
}

require_root() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    return 0
  fi
  [[ "${EUID}" -eq 0 ]] || {
    printf '需要 root 权限安装 nftables\n' >&2
    exit 1
  }
}

detect_os_family() {
  OS_FAMILY=""
  if [[ -r /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    case "${ID:-}" in
      debian | ubuntu)
        OS_FAMILY="debian"
        ;;
      centos | rocky | almalinux | rhel)
        OS_FAMILY="rhel"
        ;;
      *)
        case " ${ID_LIKE:-} " in
          *" debian "*)
            OS_FAMILY="debian"
            ;;
          *" rhel "* | *" fedora "*)
            OS_FAMILY="rhel"
            ;;
        esac
        ;;
    esac
  fi
  [[ -n "${OS_FAMILY}" ]] || {
    printf '无法识别系统类型，不能自动安装 nftables\n' >&2
    exit 1
  }
}

install_nftables() {
  detect_os_family
  case "${OS_FAMILY}" in
    debian)
      export DEBIAN_FRONTEND=noninteractive
      run_cmd apt-get update -y
      run_cmd apt-get install -y nftables
      ;;
    rhel)
      local pm="yum"
      if command -v dnf >/dev/null 2>&1; then
        pm="dnf"
      fi
      run_cmd "${pm}" install -y nftables
      ;;
  esac
}

enable_nftables_service() {
  if command -v systemctl >/dev/null 2>&1 || [[ "${DRY_RUN}" == "1" ]]; then
    run_cmd systemctl enable nftables
    run_cmd systemctl start nftables
  fi
}

require_root
if ! command -v nft >/dev/null 2>&1; then
  install_nftables
fi
enable_nftables_service

if [[ "${DRY_RUN}" != "1" ]] && ! command -v nft >/dev/null 2>&1; then
  printf 'nftables 不可用: 未找到 nft 命令\n' >&2
  exit 1
fi

printf 'nftables 可用: %s\n' "$(command -v nft || printf nft)"
