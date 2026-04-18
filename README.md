# RepoForge

Small Go service that stores uploaded artifacts on disk, tracks metadata in **SQLite**, exposes a **REST API** and a **web UI**, and rebuilds **APT** and **RPM** repository indexes so clients can consume repositories over plain HTTP.

## Screenshots

Web dashboard (repositories, uploads, health, bearer token):

![Repoforge web UI](screenshots/screenshot.png)

## Version

The current release is **0.4.2**. See [CHANGELOG.md](CHANGELOG.md) for release history. This project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html) and [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions.

## Requirements

- Go 1.22+
- **Node.js 20+** and **npm** only if you build the UI from source (see **Run**).
- For **RPM** repositories: `createrepo_c` on `PATH` (or set `CREATEREPO_C_PATH`). Used after each `.rpm` upload.
- For **DEB** repositories: no external indexer is required; `.deb` metadata is parsed in-process and `Packages` / `Release` files are generated under `dists/`.

## Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `LISTEN` | `:8080` | HTTP listen address |
| `DATA_DIR` | `./data` | Filesystem root for repository payloads |
| `DB_PATH` | `./repoforge.db` | SQLite database file |
| `REPOFORGE_TOKEN` | _(empty)_ | If set, required as `Authorization: Bearer …` for all `/v1/*` routes. **`.deb` / `.rpm` first install** creates `/etc/repoforge.env` with a random hex token (via `openssl` or `sha256sum`+`dd`) unless neither is available. |
| `CREATEREPO_C_PATH` | `createrepo_c` | Binary used for RPM metadata |
| `MAX_UPLOAD_BYTES` | `536870912` | Max upload size (bytes) |

## Run

Build the embedded UI (outputs to `internal/httpapi/webui/dist/`), then start the server:

```bash
( cd frontend && npm ci && npm run build )
go run ./cmd/repoforge
```

Open **http://127.0.0.1:8080/** for the dashboard (create repos, upload packages, optional host tooling install). `npm run dev` in `frontend/` proxies API calls to `VITE_DEV_API_URL` (default `http://127.0.0.1:8080`).

Health:

- `GET /healthz` — process up
- `GET /readyz` — SQLite reachable

## Install from GitHub (Linux)

Published **`.deb`** and **`.rpm`** files attach to the [latest GitHub Release](https://github.com/CloudVisionApps/RepoForge/releases/latest) when a `v*` tag is built (see [`.github/workflows/package-release.yml`](.github/workflows/package-release.yml)).

**One-liner** (needs `curl`, `python3`, and `sudo` unless you are already root; picks `.deb` on Debian/Ubuntu or `.rpm` on Fedora/RHEL-like systems, for **amd64** or **aarch64**):

```bash
curl -fsSL https://raw.githubusercontent.com/CloudVisionApps/RepoForge/main/scripts/install-latest-linux.sh | sudo bash
```

Use another GitHub repo (fork or mirror) by setting **`REPOFORGE_GITHUB_REPO`** before running:

```bash
curl -fsSL https://raw.githubusercontent.com/CloudVisionApps/RepoForge/main/scripts/install-latest-linux.sh | sudo env REPOFORGE_GITHUB_REPO=you/YourFork bash
```

The script lives in the repo as [`scripts/install-latest-linux.sh`](scripts/install-latest-linux.sh). Prefer downloading it, reviewing it, then executing:

```bash
curl -fsSL -o install-repoforge.sh https://raw.githubusercontent.com/CloudVisionApps/RepoForge/main/scripts/install-latest-linux.sh
less install-repoforge.sh
sudo bash install-repoforge.sh
```

## REST API

Create repositories:

```bash
curl -sS -X POST localhost:8080/v1/repositories \
  -H 'Content-Type: application/json' \
  -d '{"name":"My RPM","slug":"rpm-demo","type":"rpm"}'

curl -sS -X POST localhost:8080/v1/repositories \
  -H 'Content-Type: application/json' \
  -d '{"name":"My DEB","slug":"deb-demo","type":"deb","config":{"codename":"noble","component":"main","architectures":["amd64"]}}'

curl -sS -X POST localhost:8080/v1/repositories \
  -H 'Content-Type: application/json' \
  -d '{"name":"Files","slug":"files-demo","type":"file"}'
```

List and inspect:

```bash
curl -sS localhost:8080/v1/repositories
curl -sS localhost:8080/v1/repositories/deb-demo
curl -sS localhost:8080/v1/repositories/deb-demo/artifacts
```

Upload (multipart field **`file`**; optional **`path`** for `file` repos):

```bash
curl -sS -X POST localhost:8080/v1/repositories/rpm-demo/uploads -F 'file=@./example.rpm'
curl -sS -X POST localhost:8080/v1/repositories/deb-demo/uploads -F 'file=@./example_amd64.deb'
curl -sS -X POST localhost:8080/v1/repositories/files-demo/uploads -F 'file=@./notes.txt' -F 'path=notes/release-notes.txt'
```

When `REPOFORGE_TOKEN` is set, add the header to mutating/listing `/v1` calls:

```bash
curl -sS -H "Authorization: Bearer $REPOFORGE_TOKEN" localhost:8080/v1/repositories
```

### Install host packages for RPM (and optional DEB tooling)

`POST /v1/system/install-repo-tooling` runs the system package manager as **root** to install **createrepo_c** (and related RPM tooling) plus, on Debian-based systems, **dpkg-dev** for common `.deb` workflows. repoforge still builds APT indexes without `apt-ftparchive`; `dpkg-dev` is for tooling on the host.

**Requirements:** `REPOFORGE_TOKEN` must be set (this endpoint is refused otherwise), the process must have **euid 0**, and the body must be `{"confirm": true}`.

```bash
sudo env REPOFORGE_TOKEN=your-secret ./repoforge   # or run the systemd unit as root

curl -sS -X POST localhost:8080/v1/system/install-repo-tooling \
  -H "Authorization: Bearer $REPOFORGE_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"confirm":true}'
```

Supported families are detected from `/etc/os-release`: Debian/Ubuntu (apt), Fedora/RHEL-like (dnf/yum), Arch, openSUSE/SLES (zypper). Unsupported distros or package-manager failures return an error JSON payload (typically HTTP 500) with details in the `log` field when available.

## Client-facing HTTP (static tree)

Published files are served from:

`GET /repo/{slug}/{path}`

Examples after uploading `hello_amd64.deb` to `deb-demo`:

```bash
curl -sS localhost:8080/repo/deb-demo/dists/noble/main/binary-amd64/Packages
curl -sS localhost:8080/repo/deb-demo/pool/h/hello/hello_1.0_amd64.deb
```

### APT (`deb` repo)

`/etc/apt/sources.list.d/repoforge.list`:

```
deb [trusted=yes] http://YOUR_HOST:8080/repo/deb-demo noble main
```

Use `trusted=yes` only if you are not signing `Release` (this service does not sign in v1).

### DNF / YUM (`rpm` repo)

`/etc/yum.repos.d/repoforge.repo`:

```
[repoforge]
name=repoforge
baseurl=http://YOUR_HOST:8080/repo/rpm-demo/rpms
enabled=1
gpgcheck=0
```

`createrepo_c` writes `repodata/` next to the `.rpm` files under `rpms/`, so the `baseurl` should point at that directory.

### macOS / local development

`createrepo_c` is usually available on Fedora/RHEL-like systems or via a Linux container. On macOS, run the service inside a small Linux container or install compatible tooling (for example via Homebrew if available) so RPM indexing succeeds.

## Linux packages (.deb / .rpm)

On **Linux** with Go, **Node.js 20+**, `npm`, [FPM](https://github.com/jordansissel/fpm), and **`rpmbuild`** on `PATH` (Debian/Ubuntu: `apt install rpm`):

```bash
VERSION=0.4.2 ./packaging/build-packages.sh
```

Artifacts appear under `packaging/out/`. The systemd unit is `repoforge.service`; state defaults to `/var/lib/repoforge` with optional overrides in `/etc/repoforge.env` (see `packaging/repoforge.service`). On first package install, maintainer scripts create `/etc/repoforge.env` with an auto-generated `REPOFORGE_TOKEN` when `openssl` or `sha256sum`+`dd` is available; copy the value into the web UI token field or API clients, or comment that line out for an open `/v1` (development only).

## Layout on disk

Under `DATA_DIR/repos/{slug}/`:

- **file**: `files/…`
- **rpm**: `rpms/*.rpm` plus generated `rpms/repodata/`
- **deb**: `pool/…/*.deb` plus generated `dists/{codename}/{component}/binary-{arch}/Packages{,.gz}` and `dists/{codename}/Release`
