# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

### Removed

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
