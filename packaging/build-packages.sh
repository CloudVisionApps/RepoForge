#!/usr/bin/env bash
# Build .deb and .rpm using FPM (https://github.com/jordansissel/fpm).
# Requires: go, Node/npm, fpm (gem install fpm), rpmbuild (e.g. apt install rpm on Debian/Ubuntu).
# CGO (sqlite): build architecture matches the host (use arm64 runners for GOARCH=arm64).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGING="${ROOT}/packaging/staging"
OUT="${ROOT}/packaging/out"
PKG_NAME="repo-forge"

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
MAINTAINER="${MAINTAINER:-}"
DESCRIPTION="repo-forge: lightweight linux package manager"
URL="${URL:-}"

if ! command -v fpm >/dev/null 2>&1; then
	echo "fpm not found. Install with: gem install fpm" >&2
	exit 1
fi

RAW_VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo 0.0.0)}"
RAW_VERSION="${RAW_VERSION#v}"
PKG_VERSION="$(echo "$RAW_VERSION" | tr '-' '~' | sed 's/~dirty$/.dirty/')"

case "$GOARCH" in
amd64)
	FPM_ARCH_DEB=amd64
	FPM_ARCH_RPM=x86_64
	;;
arm64)
	FPM_ARCH_DEB=arm64
	FPM_ARCH_RPM=aarch64
	;;
*)
	echo "Unsupported GOARCH=$GOARCH (try amd64 or arm64)" >&2
	exit 1
	;;
esac

rm -rf "$STAGING"
mkdir -p "$STAGING/usr/bin" "$STAGING/usr/lib/systemd/system" \
	"$STAGING/usr/share/${PKG_NAME}/webui" "$STAGING/usr/share/doc/${PKG_NAME}" "$OUT"

echo "Building frontend (Vite)..."
(
	cd "$ROOT/frontend"
	npm ci
	npm run build
)

echo "Embedding UI for Go build..."
rm -rf "$ROOT/api/webui/dist"
mkdir -p "$ROOT/api/webui/dist"
cp -R "$ROOT/frontend/dist/." "$ROOT/api/webui/dist/"

echo "Building ${GOOS}/${GOARCH} binary..."
(
	cd "$ROOT"
	CGO_ENABLED=1 GOOS="$GOOS" GOARCH="$GOARCH" go build -buildvcs=false -trimpath -ldflags="-s -w" -o "$STAGING/usr/bin/${PKG_NAME}" .
)
chmod 755 "$STAGING/usr/bin/${PKG_NAME}"

cp -a "$ROOT/frontend/dist/." "$STAGING/usr/share/${PKG_NAME}/webui/"
chmod -R a+rX "$STAGING/usr/share/${PKG_NAME}/webui"

install -m 0644 "$ROOT/README.md" "$STAGING/usr/share/doc/${PKG_NAME}/README.md"
install -m 0644 "$ROOT/API.md" "$STAGING/usr/share/doc/${PKG_NAME}/API.md"
cp "${ROOT}/packaging/${PKG_NAME}.service" "$STAGING/usr/lib/systemd/system/${PKG_NAME}.service"

FPM_BASE=(
	-s dir
	-C "$STAGING"
	--name "$PKG_NAME"
	--version "$PKG_VERSION"
	--description "$DESCRIPTION"
	--vendor "$PKG_NAME"
	--after-install "${ROOT}/packaging/scripts/postinst.sh"
	--before-remove "${ROOT}/packaging/scripts/prerm.sh"
)
[[ -n "$URL" ]] && FPM_BASE+=(--url "$URL")
[[ -n "$MAINTAINER" ]] && FPM_BASE+=(--maintainer "$MAINTAINER")

echo "Building .deb -> $OUT"
fpm "${FPM_BASE[@]}" \
	--architecture "$FPM_ARCH_DEB" \
	-t deb \
	-p "${OUT}/" \
	.

echo "Building .rpm -> $OUT"
fpm "${FPM_BASE[@]}" \
	--architecture "$FPM_ARCH_RPM" \
	-t rpm \
	-p "${OUT}/" \
	.

echo "Done:"
ls -la "$OUT"
