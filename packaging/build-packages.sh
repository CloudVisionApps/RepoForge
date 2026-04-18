#!/usr/bin/env bash
# Build .deb and .rpm using FPM (https://github.com/jordansissel/fpm).
# Requires: go, Node 20+, npm, fpm (gem install fpm), rpmbuild (e.g. apt install rpm on Debian/Ubuntu).
# Pure Go build (modernc.org/sqlite); set GOARCH for cross-compilation on a capable host.
# Packages always embed the Linux binary (GOOS must be linux).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGING="${ROOT}/packaging/staging"
OUT="${ROOT}/packaging/out"
PKG_NAME="repoforge"
WEBUI_DIST="${ROOT}/internal/httpapi/webui/dist"

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"

if [[ "$GOOS" != "linux" ]]; then
	echo "error: FPM packages require GOOS=linux (got GOOS=$GOOS)" >&2
	exit 1
fi

for cmd in go node npm; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "error: required command not on PATH: $cmd" >&2
		exit 1
	fi
done

if ! command -v rpmbuild >/dev/null 2>&1; then
	echo "error: rpmbuild not on PATH (Debian/Ubuntu: apt install rpm)" >&2
	exit 1
fi
MAINTAINER="${MAINTAINER:-}"
DESCRIPTION="repoforge: REST upload service with SQLite and APT/RPM repository layouts"
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
	"$STAGING/usr/share/doc/${PKG_NAME}" "$OUT"

echo "Building web UI (Vite)..."
(
	cd "$ROOT/frontend"
	npm ci
	npm run build
)

if [[ ! -f "${WEBUI_DIST}/index.html" ]]; then
	echo "error: frontend build did not write ${WEBUI_DIST}/index.html" >&2
	exit 1
fi
if ! compgen -G "${WEBUI_DIST}/assets/*.js" >/dev/null; then
	echo "error: frontend build produced no ${WEBUI_DIST}/assets/*.js (embed would be broken)" >&2
	exit 1
fi

echo "Building ${GOOS}/${GOARCH} binary..."
(
	cd "$ROOT"
	CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -buildvcs=false -trimpath -ldflags="-s -w" -o "$STAGING/usr/bin/${PKG_NAME}" ./cmd/repoforge
)
chmod 755 "$STAGING/usr/bin/${PKG_NAME}"

install -m 0644 "$ROOT/README.md" "$STAGING/usr/share/doc/${PKG_NAME}/README.md"
install -m 0644 "$ROOT/CHANGELOG.md" "$STAGING/usr/share/doc/${PKG_NAME}/CHANGELOG.md"
install -m 0644 "$ROOT/docs/REST.md" "$STAGING/usr/share/doc/${PKG_NAME}/REST.md"
install -m 0644 "$ROOT/docs/CONFIGURATION.md" "$STAGING/usr/share/doc/${PKG_NAME}/CONFIGURATION.md"
install -m 0644 "$ROOT/docs/DEVELOPMENT.md" "$STAGING/usr/share/doc/${PKG_NAME}/DEVELOPMENT.md"
install -m 0644 "$ROOT/packaging/${PKG_NAME}.service" "$STAGING/usr/lib/systemd/system/${PKG_NAME}.service"

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
