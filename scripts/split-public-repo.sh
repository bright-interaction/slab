#!/usr/bin/env bash
# Produce the public fair-code mirror of Slab (the public name of atomicsite) at
# github.com/bright-interaction/slab, so `go install github.com/bright-interaction/slab/cmd/server@latest`
# resolves.
#
# Slab is open core. Unlike pare/flare/reactor (whose pro overlay lives OUTSIDE the
# repo behind a build tag), Slab's Enterprise Edition (multi-tenant cloud + billing)
# lives IN-tree behind the `ee` Go build tag. The OSS distribution compiles with the
# `!ee` stubs; the `ee`-tagged implementations are STRIPPED from the mirror so the
# cloud/billing source never ships. See ee/README.md for the boundary.
#
# Two transforms beyond the pare/flare/reactor pattern:
#   1. Module + org rename: github.com/bright-interaction/slab -> github.com/bright-interaction/slab
#      (the monorepo dir is still "atomicsite" pending the deferred rename).
#   2. Enterprise strip: every //go:build ee file + the cloud frontend routes.
# A hard assertion REFUSES to continue if any //go:build ee file survives the strip,
# so a missed path fails loudly instead of leaking enterprise code.
#
# Safe by default: no --push produces + checks the filtered tree and prints what it
# WOULD push. --push performs the outward mirror.
#
# NOTE: the Go build gate below does NOT build the Astro/Svelte frontend. Before an
# actual --push, also verify the frontend builds with the cloud routes stripped
# (cd frontend && bun install && bun run build).
set -euo pipefail

PUSH=0
REMOTE_URL="git@github.com:bright-interaction/slab.git"
PREFIX="atomicsite"
SPLIT_BRANCH="slab-public-split"

# Internal + enterprise paths stripped from the mirror's entire history. Paths are
# relative to atomicsite/ (the subtree split strips the prefix). Every entry that is
# a //go:build ee file is also caught by the assertion below; the frontend cloud
# routes are directory-scoped (no build tag) so they must be listed explicitly.
STRIP_PATHS=(
  docker-compose.yml                       # estate deploy compose (house proxy net)
  ee/cloud_ee.go                           # EE: cloud orchestration impl
  internal/cloud/mollie                    # EE: Mollie billing client (all files //go:build ee)
  internal/handlers/billing_ee.go          # EE: billing HTTP handlers
  internal/config/config_ee_test.go        # EE: cloud config test
  "frontend/src/routes/(auth)/cloud"       # EE: multi-tenant cloud UI
)

for arg in "$@"; do
  case "$arg" in
    --push) PUSH=1 ;;
    --remote=*) REMOTE_URL="${arg#--remote=}" ;;
    -h|--help) echo "usage: $0 [--push] [--remote=git@github.com:org/repo.git]"; exit 0 ;;
    *) echo "unknown arg: $arg" >&2; exit 2 ;;
  esac
done

command -v git-filter-repo >/dev/null 2>&1 || {
  echo "error: git-filter-repo is required (pip install git-filter-repo)." >&2; exit 1; }

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
[ -d "$PREFIX" ] || { echo "error: $PREFIX/ not found at $ROOT" >&2; exit 1; }

# Coarse pre-flight secret guard on the subtree history (defense before gitleaks;
# gitleaks on the filtered clone is the AUTHORITATIVE gate). The ignore list
# excludes obvious test/e2e fixtures (test-, e2e-, not-for-production, ...) so the
# heuristic does not false-trip on atomicsite's test JWT secrets; a real
# high-entropy credential without those markers still trips it.
if git log -p -- "$PREFIX/" \
  | grep -iE '(api[_-]?key|secret|password|bearer|private[_-]?key)[[:space:]]*[:=][[:space:]]*["'"'"']?[A-Za-z0-9/_+-]{16,}' \
  | grep -ivE 'changeme|change[_-]?me|your_|_here|example|redacted|placeholder|xxxx|base64_32_bytes|min_16_char|test[_-]|e2e[_-]|not-for-production|invitee-pw|strong-32-byte|32-bytes-x|DefaultJWTSecret' \
  | grep -q .; then
  echo "REFUSING: a possible secret appears in $PREFIX/ history. Audit before any push." >&2
  exit 1
fi

echo "Splitting $PREFIX/ subtree (history-preserving) into $SPLIT_BRANCH ..."
git branch -D "$SPLIT_BRANCH" >/dev/null 2>&1 || true
git subtree split --prefix="$PREFIX" -b "$SPLIT_BRANCH"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
CLONE="$WORK/slab-public"
echo "Cloning $SPLIT_BRANCH -> $CLONE (single-branch) ..."
git clone --quiet --single-branch --no-tags --branch "$SPLIT_BRANCH" "file://$ROOT" "$CLONE"

FR_ARGS=(); for p in "${STRIP_PATHS[@]}"; do FR_ARGS+=(--path "$p"); done
echo "Stripping internal + enterprise paths from all history: ${STRIP_PATHS[*]}"
( cd "$CLONE" && git filter-repo --force --invert-paths "${FR_ARGS[@]}" )

# Rename the module/org (atomicsite -> slab) and redact internal infra hostnames
# across ALL history (file contents + commit messages).
REDACT="$WORK/redactions.txt"
{
  echo 'github.com/bright-interaction/slab==>github.com/bright-interaction/slab'
  echo 'host==>host'
  echo 'web-proxy==>web-proxy'
} > "$REDACT"
echo "Renaming module atomicsite -> slab + redacting infra hostnames ..."
( cd "$CLONE" && git filter-repo --force --replace-text "$REDACT" --replace-message "$REDACT" )

# Defense in depth #1: every stripped path is gone.
for p in "${STRIP_PATHS[@]}"; do
  [ -e "$CLONE/$p" ] && { echo "REFUSING: stripped path '$p' still present." >&2; exit 1; }
done

# Defense in depth #2 (the moat guard): NO enterprise-tagged source may survive.
# This catches any //go:build ee file that was added but not added to STRIP_PATHS,
# so the cloud/billing source can never silently leak into the public mirror.
if grep -rl $'//go:build ee' "$CLONE" --include='*.go' 2>/dev/null | grep -q .; then
  echo "REFUSING: an enterprise (//go:build ee) file survived the strip:" >&2
  grep -rl $'//go:build ee' "$CLONE" --include='*.go' >&2 || true
  echo "Add it to STRIP_PATHS before any push." >&2
  exit 1
fi

echo "Build-checking the mirror (OSS !ee build) ..."
if command -v go >/dev/null 2>&1; then
  ( cd "$CLONE" && go build ./... ) && echo "  builds standalone: OK"
  ( cd "$CLONE" && go test -run='^$' ./... >/dev/null ) && echo "  tests compile: OK"
else
  echo "  (go not found; skipping build check)" >&2
fi

if command -v gitleaks >/dev/null 2>&1; then
  echo "Scanning mirror history for secrets (gitleaks) ..."
  if ! ( cd "$CLONE" && gitleaks detect --source . --config .gitleaks.toml --no-banner --redact >/dev/null 2>&1 ); then
    echo "REFUSING: gitleaks found a secret in the mirror history:" >&2
    ( cd "$CLONE" && gitleaks detect --source . --config .gitleaks.toml --no-banner --redact ) >&2 || true
    exit 1
  fi
  echo "  no secrets in mirror history: OK"
else
  echo "  WARNING: gitleaks not installed; the secret-scan gate is SKIPPED." >&2
  [ "$PUSH" -eq 1 ] && { echo "REFUSING to --push without the gitleaks gate." >&2; exit 1; }
fi

if [ "$PUSH" -eq 0 ]; then
  echo; echo "DRY RUN. Filtered mirror ready at: $CLONE"
  echo "Would push its HEAD -> $REMOTE_URL main"
  echo "Before --push also build the frontend with cloud routes stripped:"
  echo "  ( cd '$CLONE/frontend' && bun install && bun run build )"
  trap - EXIT  # keep $WORK so the operator can inspect the dry-run tree
  exit 0
fi

echo "Pushing filtered mirror -> $REMOTE_URL main ..."
( cd "$CLONE" && git push "$REMOTE_URL" HEAD:main )
echo "Done. Cleanup: git branch -D $SPLIT_BRANCH"
