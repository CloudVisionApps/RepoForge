#!/bin/sh
set -e
# Debian: remove/purge; RPM preun: 0 = last instance removed
if [ "$1" = "remove" ] || [ "$1" = "purge" ] || [ "$1" = "0" ]; then
	if command -v systemctl >/dev/null 2>&1; then
		systemctl stop bash-vhost.service 2>/dev/null || true
		systemctl disable bash-vhost.service 2>/dev/null || true
	fi
fi
exit 0
