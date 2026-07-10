# Slab

Agent-first website builder. Single Go binary + SQLite + Astro. Ships a static `dist/` you own. Auto A+ security headers, GDPR-ready cookie banner embedded, agent API for AI-driven content edits.

You self-host it for one root domain at a time (`example.com` runs Slab for `example.com`'s marketing site). Every customer-built site is just static files: serve from anywhere.

## Why

Most CMSes assume the cloud is the product. Slab assumes the dist directory is. Run the binary on a 5 EUR VPS, ship the output to Vercel, Netlify, R2, S3, or any nginx behind your firewall. The agent API lets your AI of choice (Claude, OpenAI, local llama, anything that speaks HTTP) author content, blocks, settings, and pages without ever touching the dashboard.

## 60-second quickstart (local dev)

```bash
git clone <this-repo>
cd slab

# 1. backend (port 8080)
make dev

# 2. frontend (separate terminal, port 5173 with proxy)
make frontend-install
make frontend-dev

# 3. open http://localhost:5173 and log in with the seeded admin
#    (default email: admin@slab.dev, default password: changeme123)
```

Production builds embed the SvelteKit SPA into the Go binary:

```bash
make build           # frontend → cmd/server/frontend/build → bin/slab
./bin/slab     # serves admin + tracking + agent API on :8080
```

## Deployment matrix

| Target              | OSS Core | Notes                                                                      |
|---------------------|----------|----------------------------------------------------------------------------|
| Plain VPS + nginx   | yes      | The reference shape. `bin/slab` + a per-site nginx vhost.            |
| Docker Compose      | yes      | See `docker-compose.example.yml`.                                          |
| Fly.io              | yes      | See `fly.toml`. Pre-issue TLS via Fly's edge.                              |
| Render / Railway    | yes      | Standard buildpack: `make build`, run `bin/slab`.                    |
| Kubernetes          | yes      | Stateless app, single PVC for `/data` (SQLite + uploaded media + fonts).   |
| Dockyard            | yes      | Bright Interaction's own platform. Standard container deploy.              |
| Cloudflare Pages /  | yes      | For the **built sites** (the dist output), not the admin server itself.    |
| Vercel / Netlify    |          | Admin server runs anywhere; static output ships to the edge of choice.     |
| Slab Cloud   | commercial | Hosted multi-tenant SaaS. Adds on-demand TLS, billing, SLA, support. Reach out if you'd rather not run infra. |

## Architecture

```
  ┌─────────────────────┐   ┌─────────────────┐
  │  Admin SPA          │   │  Agent API      │
  │  (SvelteKit, Bun)   │   │  (HTTP + MCP)   │
  └──────────┬──────────┘   └────────┬────────┘
             │                       │
             └───────────┬───────────┘
                         │
            ┌────────────▼────────────┐
            │  slab (Go binary) │
            │  ┌───────────────────┐  │
            │  │ chi router        │  │
            │  │ sqlc / SQLite     │  │
            │  │ builder (Astro)   │  │
            │  │ analytics tail    │  │
            │  │ CookieProof embed │  │
            │  │ DuckDB rollups    │  │
            │  └───────────────────┘  │
            └────┬───────────────┬────┘
                 │               │
                 ▼               ▼
           ┌──────────┐     ┌─────────────┐
           │ /data    │     │ dist/       │
           │ SQLite + │     │ static site │
           │ media +  │     │ → any host  │
           │ fonts    │     └─────────────┘
           └──────────┘
```

## Configuration

Copy `.env.example` to `.env` and fill in values. The minimum for a working production deploy:

| Var                            | Why                                                         |
|--------------------------------|-------------------------------------------------------------|
| `BASE_URL`                     | Public URL of the admin server (must be HTTPS in prod).     |
| `JWT_SECRET`                   | Session signing. `openssl rand -hex 32`.                    |
| `ANALYTICS_SALT`               | Visitor fingerprint salt. `openssl rand -hex 32`.           |
| `SLAB_PRIMARY_DOMAIN`    | The apex domain this instance serves (`example.com`).       |
| `ADMIN_PASSWORD`               | Initial admin login. Min 12 chars. Change before exposing.  |

The server refuses to start if any of these are unset or still equal their documented defaults in non-localhost deployments. Local dev keeps lenient defaults so contributors can run with a single `make dev`.

Full env-var reference: see `.env.example`.

## Open Core boundary

Slab ships as **Open Core (fair-code)**:

- **Core** (this repo, Slab Sustainable Use License): everything you need to self-host one Slab instance for one root domain. Go server, SQLite, builder, agent API, CookieProof embed, evaluation engine, retention manager, DuckDB analytics rollups, all the security defaults. Fair-code: read, run, modify, and self-host for free, including for your own clients. The one limit is you may not resell it or run it as a hosted service for third parties.
- **Cloud** (commercial, hosted by Bright Interaction): the enterprise (`ee`) multi-tenant edge with on-demand TLS, tenant subscription billing, cross-tenant aggregation, cert pre-issuance queue, SLA-grade backups and DR.

The Cloud layer is the held-back enterprise (`ee`) build overlay. Its real implementation is not published in this repository; the public core compiles with the `!ee` no-op stubs (`go build ./...`). The boundary and the stub seam are documented in `ee/README.md`. If you want the hosted Cloud or a commercial license, reach out at licensing@brightinteraction.com (see [LICENSING.md](LICENSING.md)).

If you self-host the Core and outgrow it, exporting your data and migrating to the Cloud is a `pg_dump`-style operation. There's no lock-in either way.

## Building from source

Requirements:
- Go 1.26+ (CGO required: the analytics rollups use DuckDB)
- Bun 1.3+ (frontend, no exceptions)
- `sqlc` (only if you change SQL)
- A C toolchain (`build-essential` on Debian/Ubuntu, `gcc + g++ + libstdc++` on Alpine)

```bash
make frontend         # builds the SvelteKit admin SPA into cmd/server/frontend/build
make build            # builds the Go binary (embeds the frontend)
go test ./...         # runs the full Go test suite
go vet ./...          # static analysis
```

The default build is the fair-code core (the `!ee` stubs). The enterprise (`ee`) cloud control plane is held back and not published here, so `-tags ee` is not buildable from this repository; the `ee/` stubs are present only to document the seam (see `ee/README.md`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). TL;DR: conventional commits, table-driven tests, no new env var without a `.env.example` entry, cloud-only code goes in `ee/`.

## Security

Slab ships with security-first defaults: A+ headers, locked-down CSP, HttpOnly + Secure + SameSite=Strict session cookies, refuse-to-start on weak secrets, salted visitor fingerprints rather than IP storage. Report vulnerabilities to security@brightinteraction.com (please do not file public issues for security bugs).

## License

Slab Sustainable Use License (fair-code). See [LICENSE](LICENSE) and [LICENSING.md](LICENSING.md). Self-host free, no reselling as a hosted service; the enterprise (`ee`) cloud layer is held back under a separate commercial license.
