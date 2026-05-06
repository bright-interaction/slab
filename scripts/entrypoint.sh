#!/bin/sh
# Atomic Site container entrypoint (Phase 30.4, 2026-05-06).
#
# When LITESTREAM_REPLICA_URL is set, the server runs under litestream
# replicate so the SQLite WAL streams continuously to the configured
# replica (Hetzner Object Storage by default). Litestream forwards
# SIGTERM to the server on shutdown so the final WAL frames are
# flushed before the container exits.
#
# When the var is empty (local dev, OSS single-deploy without backups),
# the server runs directly with no litestream wrapping. No replication,
# no cost, no extra moving parts.
#
# The litestream.yml file lives at /app/litestream.yml, copied in by
# the Dockerfile.

set -e

if [ -n "${LITESTREAM_REPLICA_URL}" ]; then
    # Restore from the replica before booting if the local DB doesn't
    # exist yet. Idempotent: if the DB is already present, restore is
    # a no-op. This is the recovery path: a fresh container started
    # against an existing replica picks up where the last one left off.
    if [ ! -f "/app/data/atomicsite.db" ]; then
        echo "[entrypoint] No local DB; restoring from ${LITESTREAM_REPLICA_URL}"
        litestream restore -if-replica-exists -config /app/litestream.yml /app/data/atomicsite.db || true
    fi
    echo "[entrypoint] Starting server under litestream replicate"
    exec litestream replicate -config /app/litestream.yml -exec "/app/server"
else
    echo "[entrypoint] LITESTREAM_REPLICA_URL unset; starting server directly"
    exec /app/server
fi
