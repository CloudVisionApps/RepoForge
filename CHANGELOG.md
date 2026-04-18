# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

- Packaging: FPM build script, systemd unit, and maintainer scripts aligned with `repoforge` (removed stale frontend / `repo-forge` / references).

### Removed

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
