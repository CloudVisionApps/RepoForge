# Configuration

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `LISTEN` | `:8080` | HTTP listen address |
| `DATA_DIR` | `./data` | Filesystem root for repository payloads |
| `DB_PATH` | `./repoforge.db` | SQLite database file |
| `REPOFORGE_TOKEN` | _(empty)_ | If set, required as `Authorization: Bearer …` for all `/v1/*` routes. |
| `CREATEREPO_C_PATH` | `createrepo_c` | Binary used for RPM metadata |
| `MAX_UPLOAD_BYTES` | `536870912` | Max upload size (bytes) |

## Linux packages (`systemd`)

The systemd unit is installed as **`/usr/lib/systemd/system/repoforge.service`**. In this repository the source file is `packaging/repoforge.service`. The unit sets `LISTEN`, `DATA_DIR`, and `DB_PATH`, and loads optional overrides from **`/etc/repoforge.env`** (`EnvironmentFile=-/etc/repoforge.env`).

On **first install** of the `.deb` or `.rpm`, maintainer scripts may create `/etc/repoforge.env` with an auto-generated **`REPOFORGE_TOKEN`** (via `openssl rand -hex 32`, or a SHA-256–based fallback if OpenSSL is unavailable) unless neither generator is available. Copy that value into the web UI token field or API clients, or remove/comment the line for an open `/v1` (not recommended outside local development).

State defaults to `/var/lib/repoforge` (see the unit file for exact paths).
