# `ee/` cloud (Enterprise Edition) build boundary

This directory holds the cloud-only code paths, gated behind the Go build tag `ee`. The OSS distribution compiles with the `!ee` stubs in `cloud_oss.go`. Setting `-tags ee` swaps in the real implementations from `cloud_ee.go`.

## What lives here

Anything that only matters when Atomic Site is hosting **multiple unrelated tenants** as a SaaS, rather than serving a single root domain for a single operator:

- Multi-tenant edge orchestration (Vercel API client, Caddy admin client)
- Domain verification poller (DNS challenge, ownership checks)
- Cert pre-issuance queue (ACME / TLS automation across many tenants)
- Billing integration (Stripe, Paddle, invoice generation)
- Cross-tenant aggregation views (rollups across tenants the operator owns)
- Cloud-specific retention, DR, and backup orchestration

## What does NOT live here

Anything the single-deployment-per-customer operator needs. That stays in `internal/` under the OSS license:

- The Go admin server, sqlc + SQLite, agent API, MCP tools
- The Astro builder + per-site dist output
- The CookieProof embed
- Auto A+ security defaults, IMY/GDPR/CCPA cookie banner stack
- The per-site nginx.conf builder, the Caddy/Netlify `_headers` writer
- The screenshot tool, eval engine, retention manager
- DuckDB analytics rollups (single-tenant scope)

## Build tags

Two files, one interface:

```
ee/cloud.go            // shared types and the Provider interface
ee/cloud_oss.go        // //go:build !ee, no-op stubs
ee/cloud_ee.go         // //go:build ee,  real implementations (or stubs that compile)
```

Build the OSS variant (default):

```bash
go build ./...
```

Build the cloud variant:

```bash
go build -tags ee ./...
```

The cloud variant currently links the same stubs as OSS. As real cloud features land, they replace the stub bodies in `cloud_ee.go` (or get pulled in from a private cloud repo via Go modules).

## Why a build tag and not a runtime flag

Both. The build tag controls **what code is linked into the binary**: a stock OSS build can't accidentally instantiate a Stripe client or a Vercel API caller, no matter what env vars are set. The `ATOMICSITE_DEPLOYMENT_MODE` runtime flag (in `internal/config`) controls **which behaviour is selected** at startup, and the `Validate()` guard refuses to start in `cloud` mode unless `ee.IsAvailable()` returns true (i.e. the binary was actually built with `-tags ee`).

This keeps the OSS distribution provably free of the closed-source surface area while letting operators who want to self-build the cloud variant do so without forking core.

## Why Apache 2.0 covers this directory too

The build-tag stubs ARE Apache 2.0. They define the interface. Any real cloud implementations Bright Interaction publishes here are also Apache 2.0. The proprietary value of Atomic Site Cloud is in the operation (multi-tenant infra, on-call, SLA, billing relationships), not in the code. If you self-build with `-tags ee` and run it for your own multi-tenant SaaS, you're not violating any license, you're just running unsupported software with no commercial backing.
