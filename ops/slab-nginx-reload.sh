#!/usr/bin/env bash
# slab-nginx-reload, invoked by the slab container's
# domain reconciler whenever a per-domain nginx fragment changes.
#
# Why a wrapper script: the container runs unprivileged and must not
# have direct sudo access. The host operator drops this script at
# /usr/local/sbin/slab-nginx-reload (root-owned, 0755) and
# whitelists it in /etc/sudoers.d/slab so the unprivileged
# slab user can call exactly this command and nothing else.
#
# Sample sudoers line (in /etc/sudoers.d/slab, mode 0440):
#
#   slab ALL=(root) NOPASSWD: /usr/local/sbin/slab-nginx-reload
#
# And in slab's environment:
#
#   SLAB_NGINX_RELOAD_CMD=/usr/bin/sudo /usr/local/sbin/slab-nginx-reload
#
# The script validates the config first ('nginx -t') and only reloads
# on success, so a bad fragment never takes down the live site.

set -euo pipefail

if ! /usr/sbin/nginx -t 2>&1; then
    echo "slab-nginx-reload: nginx -t failed; refusing to reload" >&2
    exit 1
fi

/usr/bin/systemctl reload nginx
echo "slab-nginx-reload: reload ok"
