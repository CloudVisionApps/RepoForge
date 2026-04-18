# Development

## Requirements

- Go 1.22+
- **Node.js 20+** and **npm** only if you build the UI from source (see **Run from source**).
- For **RPM** repositories: `createrepo_c` on `PATH` (or set `CREATEREPO_C_PATH`). Used after each `.rpm` upload.
- For **DEB** repositories: no external indexer is required; `.deb` metadata is parsed in-process and `Packages` / `Release` files are generated under `dists/`.

## Run from source

Build the embedded UI (outputs to `internal/httpapi/webui/dist/`), then start the server:

```bash
( cd frontend && npm ci && npm run build )
go run ./cmd/repoforge
```

Open **http://127.0.0.1:8080/** for the dashboard (create repos, upload packages, optional host tooling install).

### Vite dev server

`npm run dev` in `frontend/` proxies API calls to `VITE_DEV_API_URL` (default `http://127.0.0.1:8080`).

### Health endpoints

- `GET /healthz` — process up
- `GET /readyz` — SQLite reachable

## RPM metadata on macOS

`createrepo_c` is usually available on Fedora/RHEL-like systems or via a Linux container. On macOS, run the service inside a small Linux container or install compatible tooling (for example via Homebrew if available) so RPM indexing succeeds.

## Linux packages from source (FPM)

On **Linux** with Go, **Node.js 20+**, `npm`, [FPM](https://github.com/jordansissel/fpm), and **`rpmbuild`** on `PATH` (Debian/Ubuntu: `apt install rpm`):

```bash
VERSION=0.4.2 ./packaging/build-packages.sh
```

Artifacts appear under `packaging/out/`. See [CONFIGURATION.md](CONFIGURATION.md) for systemd paths and `/etc/repoforge.env`.

## Layout on disk

Under `DATA_DIR/repos/{slug}/`:

- **file**: `files/…`
- **rpm**: `rpms/*.rpm` plus generated `rpms/repodata/`
- **deb**: `pool/…/*.deb` plus generated `dists/{codename}/{component}/binary-{arch}/Packages{,.gz}` and `dists/{codename}/Release`
