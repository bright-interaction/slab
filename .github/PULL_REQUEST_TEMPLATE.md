<!--
Thanks for sending a PR. The checklist below mirrors CONTRIBUTING.md;
tick the boxes that apply.
-->

## What this changes

<!-- Two or three sentences. The "why" matters more than the "what" --
> the diff already shows the what. -->

## How it was tested

<!-- Concrete steps. "ran the tests" is not enough; reviewers want to
know which scenario you exercised. -->

## Checklist

- [ ] `go vet ./...` clean
- [ ] `go test ./... -race -count=1` green
- [ ] `cd frontend && bunx svelte-check --tsconfig ./tsconfig.json` reports `0 ERRORS`
- [ ] `make build` succeeds (frontend embeds, Go binary links)
- [ ] If you added an env var: documented in `.env.example` and read via `internal/config`
- [ ] If you added a SQL query: `sqlc generate` is committed
- [ ] If you changed agent / MCP surface: tool descriptions match handler behaviour
- [ ] No em dashes in user-facing copy or commit messages
- [ ] CHANGELOG.md `[Unreleased]` updated for any user-visible change

## Open Core boundary

- [ ] My change is OSS-safe (lives under `internal/`) **OR**
- [ ] My change is cloud-only and lives under `ee/` behind `//go:build ee`

## Related

<!-- Linked issues, related PRs, prior discussions. -->
