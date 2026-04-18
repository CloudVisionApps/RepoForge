#!/bin/sh
set -e

install -d -m 0755 /var/lib/repoforge/data

if [ ! -f /etc/repoforge.env ]; then
	umask 077
	printf '%s\n' '# repoforge — optional environment (root:root, chmod 600)' > /etc/repoforge.env
	printf '%s\n' '# LISTEN=:8080' >> /etc/repoforge.env
	printf '%s\n' '# DATA_DIR=/var/lib/repoforge/data' >> /etc/repoforge.env
	printf '%s\n' '# DB_PATH=/var/lib/repoforge/repoforge.db' >> /etc/repoforge.env
	printf '%s\n' '# REPOFORGE_TOKEN=generate-a-long-random-secret' >> /etc/repoforge.env
	printf '%s\n' '# CREATEREPO_C_PATH=createrepo_c' >> /etc/repoforge.env
	chmod 600 /etc/repoforge.env
	chown root:root /etc/repoforge.env 2>/dev/null || true
fi

if command -v systemctl >/dev/null 2>&1; then
	systemctl daemon-reload || true
	systemctl enable repoforge.service || true
	systemctl restart repoforge.service 2>/dev/null || systemctl start repoforge.service || true
fi
exit 0
