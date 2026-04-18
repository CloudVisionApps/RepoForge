# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

- **Docs**: development, packaging-from-source, and configuration content moved from README to [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) and [docs/CONFIGURATION.md](docs/CONFIGURATION.md); packages ship these under `/usr/share/doc/repoforge/`.

### Fixed

### Removed

## [0.4.2] - 2026-04-18

### Changed

- **Packaging**: first install now writes `/etc/repoforge.env` with an auto-generated `REPOFORGE_TOKEN` (64 hex chars from `openssl rand -hex 32`, or SHA-256 of random bytes if OpenSSL is missing) instead of a commented placeholder.

### Fixed

- **Web UI**: upload form called `reset()` on a React event after `await`, so `currentTarget` was null — capture the `<form>` before awaiting.
- **Web UI**: “Open index URL” for RPM repos no longer points at a fake `your-package.rpm`; it uses `rpms/repodata/repomd.xml`. File repos link to the first uploaded artifact when available.

## [0.4.1] - 2026-04-18

### Changed

- `packaging/build-packages.sh` requires `GOOS=linux`, checks for `go`/`node`/`npm`/`rpmbuild`, and verifies Vite output under `internal/httpapi/webui/dist/` before `go build` so packages are not built without embedded UI assets.
- **Package Release** GitHub Actions workflow runs `go test ./...` before FPM package builds.

### Fixed

- HTTP router: use chi trailing `/*` wildcards for the embedded web UI and `/repo/{slug}/…` so multi-segment paths (for example `/assets/*.js` and pool paths) are not 404. The previous `/{path:*}` pattern only matched a single segment, which broke styles/scripts and deep repository files.

## [0.4.0] - 2026-04-18

### Changed

- **Web UI**: full visual redesign — dark theme and design tokens, sidebar shell with collapsible bearer token panel, Outfit + JetBrains Mono typography, reorganized dashboard (status chips, repo cards, forms), timeline-style client docs, responsive layout for narrow viewports.

## [0.3.0] - 2026-04-18

### Added

- **Web UI** (`frontend/`): Vite + React dashboard for repositories, uploads, health, and host tooling install; production assets embedded under `GET /{path}` (SPA) from `internal/httpapi/webui/dist`.
- `GET /v1/repositories/{slug}/artifacts` — list artifacts for a repository (JSON).

### Changed

- FPM **packaging** and **GitHub Actions** again run `npm ci` / `npm run build` before `go build` so binaries include the UI.

### Fixed

- Dashboard artifact download link used a broken `href` expression.

## [0.2.0] - 2026-04-18

### Added

- `POST /v1/system/install-repo-tooling` — privileged endpoint (requires `REPOFORGE_TOKEN`, root, and `{"confirm":true}`) to install distro packages for `createrepo_c` and related repo tooling (`internal/sysrepo`).

### Fixed

- Packaging: FPM build script, systemd unit, and maintainer scripts aligned with `repoforge` (removed stale frontend / `repo-forge` / references).

## [0.1.0] - 2026-04-18

### Added

- HTTP service (`cmd/repoforge`) with configurable listen address and data directory.
- SQLite persistence with goose migrations: `repositories`, `artifacts`, `index_runs`.
- REST API under `/v1`: create and list repositories, repository detail, multipart uploads.
- Optional bearer auth for all `/v1/*` routes via `REPOFORGE_TOKEN`.
- Repository types: `deb`, `rpm`, `file` with JSON config (DEB: codename, component, architectures).
- Static publication tree at `GET /repo/{slug}/{path}` with path traversal checks.
- Health endpoints: `GET /healthz`, `GET /readyz` (database ping).
- DEB indexing in-process: AR + control archive parsing, `Packages` / `Packages.gz` per arch, `Release` with checksums; pool layout under `pool/`.
- RPM indexing via external `createrepo_c` (`CREATEREPO_C_PATH`); metadata under `rpms/repodata/`.
- Upload streaming with SHA-256 and configurable `MAX_UPLOAD_BYTES`.
