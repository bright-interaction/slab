# Licensing and open core

Slab is open core (fair-code).

## Core (this repository): Slab Sustainable Use License

Everything in this repository is licensed under the Slab Sustainable Use
License (see [LICENSE](LICENSE)). That is the whole single-operator website
builder: the Go admin server, SQLite, the Astro site builder and per-site
`dist/` output, the agent HTTP + MCP surfaces, the CookieProof consent embed,
the evaluation engine, the retention manager, the DuckDB analytics rollups,
the security defaults (A+ headers, refuse-to-start on weak secrets, salted
fingerprints), and the MCP-boundary Shield tokenization.

This is a [fair-code](https://faircode.io) license, not an OSI "open source"
license. The one limit: you may not resell Slab or run it as a hosted
service for third parties (a competing "Slab cloud"). Self-hosting,
internal commercial use, and building or hosting sites for your own clients (as
an agency or freelancer) are all expressly fine.

## Enterprise overlay (not in this repository)

A separate commercial enterprise (`ee`) build overlay (Go build tag `ee`) is
held back for the hosted SaaS and the features that only make sense at
multi-tenant scale: the multi-tenant edge orchestration, on-demand TLS and cert
pre-issuance across many tenants, tenant subscription billing, and cross-tenant
aggregation. That layer is not part of this repository. The core builds and runs
fully on its own without it (the default `go build` links the `!ee` stubs).

## Commercial license

If you want to do something the Sustainable Use License does not permit (for
example, offering Slab as a hosted service to third parties, or embedding
it in a closed product), a commercial license is available at
licensing@brightinteraction.com.
