# Personalization

Atomic Site supports light personalization: small per-visitor variations on the static site without breaking the no-PII boundary. The agent can configure rules that the build pipeline emits as runtime hooks; it cannot, ever, read identified-tier visitor data.

## What you can personalize

- A single hero variant by UTM source: `?utm_source=linkedin` swaps headline and CTA copy
- Cookie banner language by `Accept-Language` header
- A discount badge by referrer host
- A locale-default footer link by GeoIP region (when the operator wires the GeoIP layer)

## What you cannot personalize

The agent cannot:

- Read any individual visitor's history, identity, or session
- Branch on `email`, `name`, `phone`, `lead_score`, `lifecycle_stage`, or any field on `visit_sessions.metadata_json`
- Enumerate visitor IDs or correlate fingerprints across sessions
- Trigger an email or webhook based on a specific visitor's actions

The MCP server has no tool or resource that exposes identified-tier records. The personalization rules the agent writes operate on anonymous request signals only.

## How a rule is shaped

A rule has three parts:

1. **Match**: a predicate against request signals (UTM, referrer, language header, geo)
2. **Target**: a block ID and the field to override
3. **Variant**: the override value

Stored in the `personalization_rules` table; surfaced via `atomicsite://site/context.personalization` on read; written via the agent endpoint that owns this surface.

## Where it runs

`internal/builder/personalization.go` emits a small JS shim per page that reads the request signals from a server-rendered initial state, applies any matching rule, and swaps the targeted field's content. The shim runs after first paint with no layout shift; CSS hides the variant containers until a match resolves.

For high-priority swaps (LCP-affecting hero copy), do not use personalization. Bake the most likely default into the page and accept that variants flash. Personalization is correctness for tail cases, not optimisation for the modal visitor.

## Identity max-age

`analytics.identity_max_age_days` controls how long an anonymous visitor's local-storage signals persist. Default is 30 days. The agent can adjust via `bulk_upsert_settings`. Setting it to 1 effectively disables persistent personalization; setting it to 365 makes returning-visitor behaviours stick across a year.

## Do-not-touch boundary

When the operator's CRM webhook (`analytics.crm_webhook_url`) pushes visitor identification into BrightCRM, that identity flows back into `visit_sessions.metadata_json`. That column is admin-only and the MCP layer never reads it. If a request that looks like personalization needs identified data ("show different copy to known leads"), the answer is: the agent does not have access to that signal, and we will not add it. Direct the user to the admin UI's segmentation tools instead.

## Why this discipline matters

Atomic Site's privacy posture is the product. The MCP boundary that blocks identified-tier reads is enforced by `internal/mcp/mcp_test.go` and survives every refactor. An agent that quietly leaks lead-score into a page risks every customer's compliance posture. The boundary is non-negotiable.
