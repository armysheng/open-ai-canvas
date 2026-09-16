#!/bin/sh
set -eu

# Install in /etc/letsencrypt/renewal-hooks/deploy/ with mode 755.
# Only reload for this site's successful certificate renewal.
[ "${RENEWED_LINEAGE:-}" = /etc/letsencrypt/live/movie.iaigc.fun ] || exit 0
/usr/sbin/nginx -t
/usr/bin/systemctl reload nginx
