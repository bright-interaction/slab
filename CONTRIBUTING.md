# Contributing to Slab

Thanks for your interest. This file covers what you need to send a useful patch.

## Code of Conduct

Be respectful. Don't be a jerk. Harassment, personal attacks, and discrimination get you removed from the project. We follow the spirit of the [Contributor Covenant](https://www.contributor-covenant.org/), keep it simple.

## Dev setup

Requirements:
- **Go 1.26+** with CGO enabled (analytics rollups use DuckDB)
- **Bun 1.3+** for the frontend (no exceptions: never npm, never pnpm, never yarn)
- **sqlc** if you touch SQL files
- A C toolchain (`build-essential` on Debian/Ubuntu, `gcc + g++ + libstdc++` on Alpine)
- Optional: **Playwright** if you run the E2E suite (`tests/e2e/`)

First-time setup:

```bash
git clone <fork-url>
cd slab
make frontend-install     # bun install in frontend/
cp .env.example .env      # fill in JWT_SECRET, ANALYTICS_SALT for prod-ish testing

# Two terminals:
make dev                  # backend on :8080
make frontend-dev         # SvelteKit dev server with proxy on :5173
```

## Things that should never land in a commit

- **Anything inside `data/`.** That directory is the local SQLite +
  workspace cache. It is in `.gitignore` and `.gitattributes
  export-ignore`. Don't `git add -f` past it, and don't put fixture
  data there to make tests pass.
- **Real secrets.** `.env` is gitignored; commit `.env.example` only,
  with placeholder values such as `change-me-in-production`.
- **Em dashes** in code, copy, or commit messages. A pre-commit hook
  blocks them. Use a hyphen, parentheses, colon, or comma.

## Style

### Go
- Logging: `slog` only. Never `log` or `fmt.Println`.
- Errors: return them, don't panic. Use `fmt.Errorf` with `%w` for wrapping.
- Tests: table-driven (see `internal/config/config_test.go` for the pattern).
- Database: sqlc-generated queries. Don't write raw SQL inline.
- Run `go vet ./...` and `staticcheck ./...` before committing.

### Frontend
- Strict TypeScript. No `any` without a comment explaining why.
- Tailwind for styling.
- Bun for package management. Only `bun.lock` is tracked. `package-lock.json` is banned.
- Components live in `src/lib/components/`, routes in `src/routes/`.

### Commits
- Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.
- Small, focused, atomic. One logical change per commit.
- Never commit secrets. The `.gitignore` catches `.env`, `.env.*`, and `*.env`. Don't override it.

## Open Core boundary (read this if you touch the server)

Slab is **Open Core**. The OSS Core (this repo) covers everything needed to self-host one instance for one root domain. Multi-tenant edge orchestration, billing, cross-tenant aggregation, and cert pre-issuance live in the **Cloud** distribution behind the `ee` Go build tag.

Rule of thumb:
- **OSS Core**: anything one operator running their own marketing site needs. Goes in `internal/`.
- **Cloud**: anything that only matters when you're hosting multiple unrelated tenants. Goes in `ee/` behind `//go:build ee`. The OSS build ships with no-op stubs (`//go:build !ee`).

If you're unsure, ask in your PR. Do not add cloud-shaped code into core packages even temporarily; it makes splitting later much harder.

See `ee/README.md` for the canonical list of what counts as cloud-only.

## PR checklist

Before you push:

- [ ] `go vet ./...` clean
- [ ] `go test ./... -race` green
- [ ] `make frontend` succeeds (no TypeScript errors, no build warnings)
- [ ] If you added an env var: it's documented in `.env.example` and `internal/config/config.go` reads it via `envOr`/`envInt`/etc.
- [ ] If you added an SQL query: `sqlc generate` is committed
- [ ] If you changed the agent context API: `internal/agent/context.go` and the MCP tool descriptions are in sync
- [ ] If you added a security-relevant default: `internal/config/config.go::Validate()` rejects the unsafe value in production
- [ ] No em dashes in user-facing copy (the project uses plain `-` and casual phrasing)

## Reporting security issues

Email **security@brightinteraction.com** with details. Please do not file public GitHub issues for security bugs. We'll respond within 48 hours and coordinate disclosure.

## License grant

By contributing, you agree your contributions are licensed under the [Slab Sustainable Use License](LICENSE). Inbound = outbound (DCO-style: you certify you have the right to submit the work under this license); we don't require a separate CLA.
