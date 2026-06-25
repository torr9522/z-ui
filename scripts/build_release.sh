#!/usr/bin/env bash
# Build z-ui release package. Downloads xray + dat files into bin/ if missing.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-$(grep '^version:' "${ROOT}/config/version" | awk '{print $2}')}"
XRAY_VERSION="${XRAY_VERSION:-v25.6.8}"

archs=(amd64 arm64)

for arch in "${archs[@]}"; do
  echo "=== Building z-ui-linux-${arch}.tar.gz ==="
  pkg_dir="${ROOT}/.release/v${VERSION}/pkg-${arch}/x-ui"
  rm -rf "${pkg_dir}" && mkdir -p "${pkg_dir}/bin"

  # Copy source files first (exclude bin/, built binary, large/irrelevant dirs)
  rsync -a --exclude='bin/' --exclude='.release/' --exclude='.git/' \
    --exclude='x-ui' --exclude='node_modules/' --exclude='combined-backup/' \
    --exclude='archive/' --exclude='backup/' --exclude='pre-reboot*' \
    --exclude='*.tar.gz' --exclude='*.sha256' --exclude='*.patch' \
    --exclude='packaging/logrotate' \
    "${ROOT}/" "${pkg_dir}/"
  mkdir -p "${pkg_dir}/packaging/logrotate"
  cp "${ROOT}/packaging/logrotate/x-ui-xray-access" "${pkg_dir}/packaging/logrotate/" 2>/dev/null || true

  # Build Go binary AFTER rsync so it is not overwritten
  echo "  Building Go binary (arch=${arch})..."
  cd "${ROOT}"
  GOOS=linux GOARCH="${arch}" go build -o "${pkg_dir}/x-ui" .

  # Download xray binary if not in local bin/
  xray_src="${ROOT}/bin/xray-linux-${arch}"
  if [[ -x "${xray_src}" ]]; then
    cp "${xray_src}" "${pkg_dir}/bin/xray-linux-${arch}"
    echo "  Copied xray-linux-${arch} from local bin/"
  else
    echo "  Downloading xray ${XRAY_VERSION} for ${arch}..."
    xray_arch="${arch}"; [[ "${arch}" == "amd64" ]] && xray_arch="64" || xray_arch="arm64-v8a"
    curl -fsSL -o /tmp/xray-${arch}.zip \
      "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-${xray_arch}.zip"
    unzip -o /tmp/xray-${arch}.zip xray -d /tmp/xray-${arch}-bin/
    mv /tmp/xray-${arch}-bin/xray "${pkg_dir}/bin/xray-linux-${arch}"
    rm -rf /tmp/xray-${arch}.zip /tmp/xray-${arch}-bin/
  fi
  chmod 755 "${pkg_dir}/bin/xray-linux-${arch}"

  # Copy dat files
  for dat in geoip.dat geosite.dat; do
    if [[ -f "${ROOT}/bin/${dat}" ]]; then
      cp "${ROOT}/bin/${dat}" "${pkg_dir}/bin/${dat}"
    else
      echo "  Downloading ${dat}..."
      curl -fsSL -o "${pkg_dir}/bin/${dat}" \
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/${dat}"
    fi
  done

  # Validate
  [[ -x "${pkg_dir}/bin/xray-linux-${arch}" ]] || { echo "ERROR: xray binary missing"; exit 1; }
  [[ -s "${pkg_dir}/bin/geoip.dat" ]] || { echo "ERROR: geoip.dat empty"; exit 1; }
  [[ -s "${pkg_dir}/bin/geosite.dat" ]] || { echo "ERROR: geosite.dat empty"; exit 1; }

  # Package
  out="${ROOT}/.release/v${VERSION}/z-ui-linux-${arch}.tar.gz"
  tar -czf "${out}" -C "${ROOT}/.release/v${VERSION}/pkg-${arch}" x-ui
  echo "  Package: ${out}"
  echo "  bin/ contents:"
  tar -tzf "${out}" | grep "x-ui/bin/"
done

# SHA256SUMS
cd "${ROOT}/.release/v${VERSION}"
sha256sum z-ui-linux-amd64.tar.gz z-ui-linux-arm64.tar.gz > SHA256SUMS
cat SHA256SUMS
echo "=== Build complete: v${VERSION} ==="
