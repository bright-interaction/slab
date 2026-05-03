#!/usr/bin/env bash
# atomicsite-nginx-reload — invoked by the atomicsite container's
# domain reconciler whenever a per-domain nginx fragment changes.
#
# Why a wrapper script: the container runs unprivileged and must not
# have direct sudo access. The host operator drops this script at
# /usr/local/sbin/atomicsite-nginx-reload (root-owned, 0755) and
# whitelists it in /etc/sudoers.d/atomicsite so the unprivileged
# atomicsite user can call exactly this command and nothing else.
#
# Sample sudoers line (in /etc/sudoers.d/atomicsite, mode 0440):
#
#   atomicsite ALL=(root) NOPASSWD: /usr/local/sbin/atomicsite-nginx-reload
#
# And in atomicsite's environment:
#
#   ATOMICSITE_NGINX_RELOAD_CMD=/usr/bin/sudo /usr/local/sbin/atomicsite-nginx-reload
#
# The script validates the config first ('nginx -t') and only reloads
# on success, so a bad fragment never takes down the live site.

set -euo pipefail

if ! /usr/sbin/nginx -t 2>&1; then
    echo "atomicsite-nginx-reload: nginx -t failed; refusing to reload" >&2
    exit 1
fi

/usr/bin/systemctl reload nginx
echo "atomicsite-nginx-reload: reload ok"
