#!/usr/bin/env bash
# Install the latest repoforge .deb or .rpm from GitHub Releases (Linux only).
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/CloudVisionApps/RepoForge/main/scripts/install-latest-linux.sh | sudo bash
# Optional:
#   REPOFORGE_GITHUB_REPO=owner/OtherFork bash scripts/install-latest-linux.sh
set -euo pipefail

REPO="${REPOFORGE_GITHUB_REPO:-CloudVisionApps/RepoForge}"
API="https://api.github.com/repos/${REPO}/releases/latest"

die() {
	echo "install-latest-linux: $*" >&2
	exit 1
}

have() {
	command -v "$1" >/dev/null 2>&1
}

as_root() {
	if [[ "$(id -u)" -eq 0 ]]; then
		"$@"
	else
		if ! have sudo; then
			die "need root or sudo to install packages"
		fi
		sudo "$@"
	fi
}

if ! have python3; then
	die "python3 is required to read the GitHub API response"
fi

ARCH="$(uname -m || true)"
case "$ARCH" in
x86_64 | amd64)
	DEB_SUF="_amd64.deb"
	RPM_SUF=".x86_64.rpm"
	;;
aarch64 | arm64)
	DEB_SUF="_arm64.deb"
	RPM_SUF=".aarch64.rpm"
	;;
*)
	die "unsupported machine architecture: ${ARCH:-unknown} (need x86_64 or aarch64/arm64)"
	;;
esac

if ! have curl; then
	die "curl is required (e.g. apt install curl / dnf install curl)"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Fetching latest release from ${API} ..."
JSON="${TMP}/latest.json"
if ! curl -fsSL -o "$JSON" --connect-timeout 20 --retry 3 "$API"; then
	die "failed to download release metadata (check network and repo name: ${REPO})"
fi

pick_asset_url() {
	local kind="$1"
	local suf="$2"
	python3 - "$JSON" "$kind" "$suf" <<'PY'
import json, sys

path, kind, suf = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)
for a in data.get("assets") or []:
    name = a.get("name") or ""
    url = a.get("browser_download_url") or ""
    if url and name.endswith(suf):
        print(url)
        raise SystemExit(0)
raise SystemExit(2)
PY
}

PKG_KIND=""
if have apt-get; then
	PKG_KIND=deb
elif have dnf || have yum || have rpm; then
	PKG_KIND=rpm
else
	die "need apt-get (Debian/Ubuntu) or dnf/yum/rpm (Fedora/RHEL-like) to install the package"
fi

SUF=""
if [[ "$PKG_KIND" == deb ]]; then
	SUF="$DEB_SUF"
else
	SUF="$RPM_SUF"
fi

URL=""
if URL="$(pick_asset_url "$PKG_KIND" "$SUF")"; then
	:
else
	URL=""
fi

[[ -n "$URL" ]] || die "no matching ${PKG_KIND} asset ending with ${SUF} in the latest release (CI must publish packages for this architecture)"

NAME="${URL##*/}"
OUT="${TMP}/${NAME}"
echo "Downloading ${NAME} ..."
curl -fSL --connect-timeout 30 --retry 3 -o "$OUT" "$URL"

if [[ "$PKG_KIND" == deb ]]; then
	echo "Installing .deb with apt-get ..."
	as_root env DEBIAN_FRONTEND=noninteractive apt-get install -y "$OUT"
else
	if have dnf; then
		echo "Installing .rpm with dnf ..."
		as_root dnf install -y "$OUT"
	elif have yum; then
		echo "Installing .rpm with yum ..."
		as_root yum install -y "$OUT"
	else
		echo "Installing .rpm with rpm ..."
		as_root rpm -Uvh "$OUT"
	fi
fi

echo "Done. Try: systemctl status repoforge --no-pager"
echo "Config: /etc/repoforge.env (token may have been generated on first install)"
