# `ee/` cloud (Enterprise Edition) build boundary

This directory documents the cloud-only build boundary, gated behind the Go build tag `ee`. The public fair-code distribution compiles with the `!ee` stubs in `cloud_oss.go` (the default `go build ./...`). The real `-tags ee` implementations are the held-back commercial enterprise layer and are NOT published in this repository, so `-tags ee` is not buildable from the public core; only the interface (`cloud.go`) and the `!ee` stubs ship here to document the seam.

## What lives here

Anything that only matters when Slab is hosting **multiple unrelated tenants** as a SaaS, rather than serving a single root domain for a single operator:

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
ee/cloud.go            // shared types and the Provider interface (ships here)
ee/cloud_oss.go        // //go:build !ee, no-op stubs (ships here)
ee/cloud_ee.go         // //go:build ee,  real implementations (HELD BACK, not in this repo)
```

Build the public fair-code variant (default):

```bash
go build ./...
```

The `-tags ee` variant is the held-back commercial enterprise layer. Its real
implementations (and the tenant subscription billing, multi-tenant edge, and
cross-tenant aggregation packages it links) are not published in this
repository, so `go build -tags ee ./...` does not build from the public core.
The interface and the `!ee` stubs ship here purely to document the seam.

## Why a build tag and not a runtime flag

Both. The build tag controls **what code is linked into the binary**: a stock OSS build can't accidentally instantiate a Stripe client or a Vercel API caller, no matter what env vars are set. The `SLAB_DEPLOYMENT_MODE` runtime flag (in `internal/config`) controls **which behaviour is selected** at startup, and the `Validate()` guard refuses to start in `cloud` mode unless `ee.IsAvailable()` returns true (i.e. the binary was actually built with `-tags ee`).

This keeps the OSS distribution provably free of the closed-source surface area while letting operators who want to self-build the cloud variant do so without forking core.

## Licensing: fair-code core, held-back commercial enterprise layer

The public core of Slab (everything in this repository, including the
`ee/` interface and the `!ee` stubs) is licensed under the Slab
Sustainable Use License, a fair-code license: read, run, modify, and self-host
it for free, including for your own clients. The one limit is that you may not
resell it or run it as a hosted service for third parties. See
[../LICENSE](../LICENSE) and [../LICENSING.md](../LICENSING.md).

The real `-tags ee` code paths (multi-tenant edge orchestration, tenant
subscription billing, cross-tenant aggregation, cert pre-issuance across many
tenants) are the commercial enterprise layer. That code is held back: it is not
published in this repository and is offered under a separate commercial license.
The public core builds and runs fully on its own without it. If you want the
hosted Slab Cloud or a commercial license for the enterprise layer,
contact licensing@brightinteraction.com.
