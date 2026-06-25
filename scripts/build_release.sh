#!/usr/bin/env bash
# Build z-ui release packages with deterministic provenance metadata.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-$(grep '^version:' "${ROOT}/config/version" | awk '{print $2}')}"
XRAY_VERSION="${XRAY_VERSION:-v25.6.8}"
COMMIT="$(cd "${ROOT}" && git rev-parse HEAD)"
BRANCH="$(cd "${ROOT}" && git branch --show-current || true)"
BRANCH="${BRANCH:-unknown}"
HOST_GOOS="$(go env GOOS)"
HOST_GOARCH="$(go env GOARCH)"
BUILD_TIME="$(date -u '+%Y-%m-%d %H:%M:%S')"
MODULE_PATH="$(cd "${ROOT}" && go list -m)"
CONFIG_PKG="${MODULE_PATH}/config"

archs=(amd64 arm64)

validate_pkg_metadata() {
  local pkg_dir="$1"
  local arch="$2"
  grep -qx "version: ${VERSION}" "${pkg_dir}/config/version" || { echo "ERROR: package config/version mismatch" >&2; exit 1; }
  grep -qx "commit: ${COMMIT}" "${pkg_dir}/config/version" || { echo "ERROR: package config commit mismatch" >&2; exit 1; }
  grep -qx "build_time: ${BUILD_TIME}" "${pkg_dir}/config/version" || { echo "ERROR: package build_time mismatch" >&2; exit 1; }

  # Full runtime metadata validation is possible only for the native builder arch.
  if [[ "${HOST_GOOS}" == "linux" && "${HOST_GOARCH}" == "${arch}" ]]; then
    local version_out commit_out branch_out
    version_out="$(${pkg_dir}/x-ui version | sed -n 's/^Version: //p' | head -1)"
    commit_out="$(${pkg_dir}/x-ui version | sed -n 's/^Commit: //p' | head -1)"
    branch_out="$(${pkg_dir}/x-ui version | sed -n 's/^Branch: //p' | head -1)"
    [[ "${version_out}" == "${VERSION}" ]] || { echo "ERROR: binary version mismatch: ${version_out} != ${VERSION}" >&2; exit 1; }
    [[ "${commit_out}" == "${COMMIT}" ]] || { echo "ERROR: binary commit mismatch: ${commit_out} != ${COMMIT}" >&2; exit 1; }
    [[ "${branch_out}" == "${BRANCH}" ]] || { echo "ERROR: binary branch mismatch: ${branch_out} != ${BRANCH}" >&2; exit 1; }
  else
    grep -aq "${COMMIT}" "${pkg_dir}/x-ui" || { echo "ERROR: cross-arch binary does not contain commit ${COMMIT}" >&2; exit 1; }
    grep -aq "${BRANCH}" "${pkg_dir}/x-ui" || { echo "ERROR: cross-arch binary does not contain branch ${BRANCH}" >&2; exit 1; }
  fi
}

for arch in "${archs[@]}"; do
  echo "=== Building z-ui-linux-${arch}.tar.gz ==="
  pkg_dir="${ROOT}/.release/v${VERSION}/pkg-${arch}/x-ui"
  rm -rf "${pkg_dir}" && mkdir -p "${pkg_dir}/bin"

  rsync -a --exclude='bin/' --exclude='.release/' --exclude='.git/' \
    --exclude='x-ui' --exclude='node_modules/' --exclude='combined-backup/' \
    --exclude='archive/' --exclude='backup/' --exclude='pre-reboot*' \
    --exclude='*.tar.gz' --exclude='*.sha256' --exclude='*.patch' \
    --exclude='packaging/logrotate' \
    "${ROOT}/" "${pkg_dir}/"
  mkdir -p "${pkg_dir}/packaging/logrotate"
  cp "${ROOT}/packaging/logrotate/x-ui-xray-access" "${pkg_dir}/packaging/logrotate/" 2>/dev/null || true

  mkdir -p "${pkg_dir}/config"
  cat >"${pkg_dir}/config/version" <<VERSION_EOF
version: ${VERSION}
commit: ${COMMIT}
build_time: ${BUILD_TIME}
VERSION_EOF

  echo "  Building Go binary (arch=${arch}, commit=${COMMIT})..."
  cd "${ROOT}"
  GOOS=linux GOARCH="${arch}" go build \
    -ldflags "-X '${CONFIG_PKG}.BuildCommit=${COMMIT}' -X '${CONFIG_PKG}.BuildBranch=${BRANCH}' -X '${CONFIG_PKG}.BuildTime=${BUILD_TIME}'" \
    -o "${pkg_dir}/x-ui" .

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

  for dat in geoip.dat geosite.dat; do
    if [[ -f "${ROOT}/bin/${dat}" ]]; then
      cp "${ROOT}/bin/${dat}" "${pkg_dir}/bin/${dat}"
    else
      echo "  Downloading ${dat}..."
      curl -fsSL -o "${pkg_dir}/bin/${dat}" \
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/${dat}"
    fi
  done

  [[ -x "${pkg_dir}/bin/xray-linux-${arch}" ]] || { echo "ERROR: xray binary missing" >&2; exit 1; }
  [[ -s "${pkg_dir}/bin/geoip.dat" ]] || { echo "ERROR: geoip.dat empty" >&2; exit 1; }
  [[ -s "${pkg_dir}/bin/geosite.dat" ]] || { echo "ERROR: geosite.dat empty" >&2; exit 1; }
  validate_pkg_metadata "${pkg_dir}" "${arch}"

  out="${ROOT}/.release/v${VERSION}/z-ui-linux-${arch}.tar.gz"
  tar -czf "${out}" -C "${ROOT}/.release/v${VERSION}/pkg-${arch}" x-ui
  echo "  Package: ${out}"
  tar -tzf "${out}" | grep -E 'x-ui/bin/(xray-linux-'"${arch}"'|geoip.dat|geosite.dat)$'
done

cd "${ROOT}/.release/v${VERSION}"
sha256sum z-ui-linux-amd64.tar.gz z-ui-linux-arm64.tar.gz > SHA256SUMS
cat SHA256SUMS
echo "=== Build complete: v${VERSION} commit=${COMMIT} branch=${BRANCH} build_time=${BUILD_TIME} ==="
