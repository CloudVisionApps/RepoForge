#!/bin/sh
set -e

install -d -m 0755 /var/lib/bash-vhost

ENV_FILE=/etc/bash-vhost.env

key_configured() {
	[ -f "$ENV_FILE" ] || return 1
	line=$(grep '^BASH_VHOST_API_KEY=' "$ENV_FILE" 2>/dev/null | head -n1 || true)
	[ -n "$line" ] || return 1
	val=${line#BASH_VHOST_API_KEY=}
	val=$(printf '%s' "$val" | tr -d '\r')
	val=$(printf '%s' "$val" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	[ "${#val}" -ge 16 ]
}

gen_key() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 32
		return
	fi
	dd if=/dev/urandom bs=32 count=1 2>/dev/null | od -An -tx1 | tr -d ' \n'
}

write_env_with_key() {
	KEY=$1
	umask 077
	if [ -f "$ENV_FILE" ]; then
		tmp=$(mktemp)
		grep -v '^BASH_VHOST_API_KEY=' "$ENV_FILE" > "$tmp" || true
		printf 'BASH_VHOST_API_KEY=%s\n' "$KEY" >> "$tmp"
		mv "$tmp" "$ENV_FILE"
	else
		{
			printf '%s\n' '# bash-vhost runtime configuration (root:root, chmod 600)'
			printf '%s\n' '# API clients and the web UI need this key (X-API-Key / Bearer).'
			printf 'BASH_VHOST_API_KEY=%s\n' "$KEY"
		} > "$ENV_FILE"
	fi
	chmod 600 "$ENV_FILE"
	chown root:root "$ENV_FILE" 2>/dev/null || true
}

if ! key_configured; then
	KEY=$(gen_key)
	write_env_with_key "$KEY"
	echo "bash-vhost: generated BASH_VHOST_API_KEY in ${ENV_FILE} — use it for API and UI access." >&2
fi

if command -v systemctl >/dev/null 2>&1; then
	systemctl daemon-reload || true
	systemctl enable bash-vhost.service || true
	systemctl restart bash-vhost.service || systemctl start bash-vhost.service || true
fi
exit 0
