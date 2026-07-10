#!/usr/bin/env bash
# Produce the public fair-code mirror of Slab at
# github.com/bright-interaction/slab, so
# `go install github.com/bright-interaction/slab/cmd/server@latest` resolves.
#
# Slab is open core (fair-code). The whole atomicsite/ tree (the monorepo
# directory keeps its atomicsite/ name; only the product identity was renamed
# to Slab) ships in the mirror under the Slab Sustainable Use License
# (fair-code: self-host free,
# no reselling as a hosted service), EXCEPT the enterprise (ee) cloud layer,
# which is HELD BACK: the `-tags ee` implementations (tenant subscription
# billing, multi-tenant edge) are stripped from the mirror's entire history so
# only the `!ee` stubs (which the default `go build ./...` links) remain. See
# LICENSING.md + ee/README.md. This script also strips the estate deploy compose
# + internal CI marker and redacts internal infra references from all history,
# then secret-scans and build-checks before any push.
#
# Safe by default: with no --push it produces + checks the filtered tree and prints
# what it WOULD push. --push performs the outward mirror (requires the public repo
# to exist: gh repo create bright-interaction/slab --public).
#
# Pattern (single-branch split-clone + gitleaks gate) mirrors mesh/reactor/flare/pare;
# see the Hive gotcha "mesh-mirror-split-clone-drags-in-monorepo-branch-secrets".
set -euo pipefail

PUSH=0
REMOTE_URL="git@github.com:bright-interaction/slab.git"
# PREFIX is the monorepo DIRECTORY name, intentionally NOT renamed (the live
# deploy pipeline targets atomicsite/); only the product identity became Slab.
PREFIX="atomicsite"
SPLIT_BRANCH="slab-public-split"

# Internal-ONLY paths + the held-back enterprise (ee) layer: stripped from the
# mirror's entire history. Paths are relative to atomicsite/ (the subtree split
# strips the prefix).
#   - ee/cloud_ee.go, internal/cloud/mollie/, internal/handlers/billing_ee.go,
#     internal/config/config_ee_test.go: the //go:build ee commercial layer
#     (tenant subscription billing + the ee Provider seam). The !ee stubs
#     (ee/cloud_oss.go, internal/handlers/billing_oss.go,
#     internal/config/config_oss_test.go, ee/cloud.go) STAY so the default
#     !ee `go build ./...` compiles standalone.
#   - .hephaestus-trigger: internal CI rebuild marker, not app code.
#   - docker-compose.yml: the ESTATE production compose (BI multi-tenant edge,
#     host bind-mounts, CI-provisioned env). Public self-hosters use
#     docker-compose.example.yml instead, which is generic.
STRIP_PATHS=(
  ee/cloud_ee.go
  internal/cloud/mollie/
  internal/handlers/billing_ee.go
  internal/config/config_ee_test.go
  .hephaestus-trigger
  docker-compose.yml
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

# Coarse pre-flight secret guard on the subtree history (defense before the
# authoritative gitleaks scan on the filtered clone below). The value class
# excludes '.' so dotted Go selectors / SQL columns break into sub-16-char tokens
# instead of false-tripping; documented placeholders AND obvious test-fixture
# markers (e2e-*, test-jwt-*, DefaultJWTSecret, etc. - the JWT/secret constants
# baked into *_test.go and tests/e2e/) are filtered out. A real high-entropy
# credential is a contiguous >=16 run without those markers and still trips the
# guard; dotted secrets (JWTs) are caught by the gitleaks gate below.
if git log -p -- "$PREFIX/" \
  | grep -iE '(api[_-]?key|secret|password|bearer|private[_-]?key)[[:space:]]*[:=][[:space:]]*["'"'"']?[A-Za-z0-9/_+-]{16,}' \
  | grep -ivE 'changeme|change[_-]?me|your_|_here|example|redacted|placeholder|xxxx|base64_32_bytes|min_16_char|e2e[_-]|not-for-production|defaultjwtsecret|test[_-]jwt|test[_-]secret|strong-32-byte|invitee|crm-hmac' \
  | grep -q .; then
  echo "REFUSING: a possible secret appears in $PREFIX/ history. Audit before any push:" >&2
  echo "  git log -p -- $PREFIX/ | grep -iE 'key|secret|token|password'" >&2
  exit 1
fi

echo "Splitting $PREFIX/ subtree (history-preserving) into $SPLIT_BRANCH ..."
git branch -D "$SPLIT_BRANCH" >/dev/null 2>&1 || true
git subtree split --prefix="$PREFIX" -b "$SPLIT_BRANCH"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
CLONE="$WORK/slab-public"
# --single-branch + --no-tags: the throwaway clone holds ONLY the disjoint
# atomicsite subtree history, never the monorepo's other branches (which carry
# unrelated project CI secrets). The clone == the publish payload, which makes
# the gitleaks scan below authoritative. file:// disables the hardlink path.
echo "Cloning $SPLIT_BRANCH -> $CLONE (single-branch) ..."
git clone --quiet --single-branch --no-tags --branch "$SPLIT_BRANCH" "file://$ROOT" "$CLONE"

if [ "${#STRIP_PATHS[@]}" -gt 0 ]; then
  FR_ARGS=(); for p in "${STRIP_PATHS[@]}"; do FR_ARGS+=(--path "$p"); done
  echo "Stripping internal-only + held-back ee paths from all history: ${STRIP_PATHS[*]}"
  ( cd "$CLONE" && git filter-repo --force --invert-paths "${FR_ARGS[@]}" )
fi

# Redact internal infra references from ALL history (file contents + commit
# messages). Distinctive tokens only, so a literal global replace is safe.
# The generic company domain is NOT globally replaced: security@/licensing@/
# conduct@brightinteraction.com are the real public contacts and must stay;
# only the internal SUBDOMAINS + the personal email are mapped.
REDACT="$WORK/redactions.txt"
{
  echo 'host==>host'
  echo '203.0.113.10==>203.0.113.10'
  echo 'auth.example.com==>auth.example.com'
  echo 'cal.example.com==>cal.example.com'
  echo 'consent.example.com==>consent.example.com'
  echo 'atomicsite.example.com==>atomicsite.example.com'
  echo 'user@example.com==>user@example.com'
  echo 'the deploy pipeline==>the deploy pipeline'
  echo 'the CI system==>the CI system'
} > "$REDACT"
echo "Redacting internal infra references from all history ..."
( cd "$CLONE" && git filter-repo --force --replace-text "$REDACT" --replace-message "$REDACT" )

# Defense in depth: fail if a stripped path survived.
for p in "${STRIP_PATHS[@]}"; do
  [ -e "$CLONE/$p" ] && { echo "REFUSING: stripped path '$p' still present." >&2; exit 1; }
done

echo "Build-checking the mirror ..."
if command -v go >/dev/null 2>&1; then
  ( cd "$CLONE" && go build ./... ) && echo "  builds standalone: OK"
  ( cd "$CLONE" && go test -run='^$' ./... >/dev/null ) && echo "  tests compile: OK"
else
  echo "  (go not found; skipping build check)" >&2
fi

# Authoritative secret scan: the single-branch clone IS the publish payload.
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
  echo "  Install it before pushing: brew install gitleaks" >&2
  [ "$PUSH" -eq 1 ] && { echo "REFUSING to --push without the gitleaks gate." >&2; exit 1; }
fi

if [ "$PUSH" -eq 0 ]; then
  echo; echo "DRY RUN. Filtered mirror ready at: $CLONE"
  echo "Would push its HEAD -> $REMOTE_URL main"
  echo "Re-run with --push once the public repo exists (gh repo create bright-interaction/slab --public)."
  trap - EXIT  # keep $WORK so the operator can inspect the dry-run tree
  exit 0
fi

echo "Pushing filtered mirror -> $REMOTE_URL main ..."
( cd "$CLONE" && git push "$REMOTE_URL" HEAD:main )
echo "Done. Cleanup: git branch -D $SPLIT_BRANCH"
