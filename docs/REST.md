# repoforge REST API examples

Examples assume the server listens on **`http://localhost:8080`**. Replace the host or port as needed.

## Create repositories

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

## List and inspect

```bash
curl -sS localhost:8080/v1/repositories
curl -sS localhost:8080/v1/repositories/deb-demo
curl -sS localhost:8080/v1/repositories/deb-demo/artifacts
```

## Upload

Multipart field **`file`**; optional **`path`** for `file` repositories.

```bash
curl -sS -X POST localhost:8080/v1/repositories/rpm-demo/uploads -F 'file=@./example.rpm'
curl -sS -X POST localhost:8080/v1/repositories/deb-demo/uploads -F 'file=@./example_amd64.deb'
curl -sS -X POST localhost:8080/v1/repositories/files-demo/uploads -F 'file=@./notes.txt' -F 'path=notes/release-notes.txt'
```

## Bearer token

When **`REPOFORGE_TOKEN`** is set, send it on mutating and listing **`/v1`** calls:

```bash
curl -sS -H "Authorization: Bearer $REPOFORGE_TOKEN" localhost:8080/v1/repositories
```

## Install host packages (RPM / DEB tooling)

**`POST /v1/system/install-repo-tooling`** runs the system package manager as **root** to install **`createrepo_c`** (and related RPM tooling) plus, on Debian-based systems, **`dpkg-dev`** for common `.deb` workflows. repoforge still builds APT indexes without `apt-ftparchive`; `dpkg-dev` is optional host tooling.

**Requirements:** `REPOFORGE_TOKEN` must be set (this endpoint is refused otherwise), the process must have **euid 0**, and the body must be **`{"confirm": true}`**.

```bash
sudo env REPOFORGE_TOKEN=your-secret ./repoforge   # or run the systemd unit as root

curl -sS -X POST localhost:8080/v1/system/install-repo-tooling \
  -H "Authorization: Bearer $REPOFORGE_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"confirm":true}'
```

Supported families are detected from **`/etc/os-release`**: Debian/Ubuntu (apt), Fedora/RHEL-like (dnf/yum), Arch, openSUSE/SLES (zypper). Unsupported distros or package-manager failures return an error JSON payload (typically HTTP 500) with details in the **`log`** field when available.
