#!/bin/sh
set -e

install -d -m 0755 /var/lib/repoforge/data

if [ ! -f /etc/repoforge.env ]; then
	umask 077
	TOKEN=""
	if command -v openssl >/dev/null 2>&1; then
		TOKEN=$(openssl rand -hex 32)
	elif command -v sha256sum >/dev/null 2>&1 && command -v dd >/dev/null 2>&1; then
		TOKEN=$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | sha256sum | awk '{print $1}')
	fi
	{
		printf '%s\n' '# repoforge — optional environment (root:root, chmod 600)'
		printf '%s\n' '# LISTEN=:8080'
		printf '%s\n' '# DATA_DIR=/var/lib/repoforge/data'
		printf '%s\n' '# DB_PATH=/var/lib/repoforge/repoforge.db'
		printf '%s\n' '# CREATEREPO_C_PATH=createrepo_c'
		printf '%s\n' ''
		printf '%s\n' '# Bearer token for all /v1/* routes. Comment out or delete to allow unauthenticated /v1 (not recommended).'
		if [ -n "$TOKEN" ]; then
			printf 'REPOFORGE_TOKEN=%s\n' "$TOKEN"
		else
			printf '%s\n' '# REPOFORGE_TOKEN=install openssl or set manually'
		fi
	} >/etc/repoforge.env
	chmod 600 /etc/repoforge.env
	chown root:root /etc/repoforge.env 2>/dev/null || true
fi

if command -v systemctl >/dev/null 2>&1; then
	systemctl daemon-reload || true
	systemctl enable repoforge.service || true
	systemctl restart repoforge.service 2>/dev/null || systemctl start repoforge.service || true
fi
exit 0
