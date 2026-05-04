# Security authoring

Atomic Site emits 11 A+ security headers by default. The agent's job is to compose pages that work within those headers, and to whitelist external origins through the right surface (`allowed_scripts` for scripts, settings overrides for everything else) rather than weakening the policy.

## The eleven headers

1. `Content-Security-Policy`: locked to `'self'` plus operator-whitelisted origins
2. `Strict-Transport-Security`: HSTS with preload-eligible max-age
3. `X-Frame-Options`: `DENY` by default; per-site override via `security.frame_ancestors`
4. `X-Content-Type-Options`: `nosniff`
5. `Referrer-Policy`: `strict-origin-when-cross-origin`
6. `Permissions-Policy`: 20 directives locked off
7. `Cross-Origin-Opener-Policy`: `same-origin-allow-popups` so OAuth popups work
8. `Cross-Origin-Resource-Policy`: `same-origin`
9. `Cross-Origin-Embedder-Policy`: `unsafe-none` (relaxed; tightening breaks third-party iframes)
10. `X-XSS-Protection`: `1; mode=block` (legacy but Site Inspector still grades for it)
11. `X-Permitted-Cross-Domain-Policies`: `none`

The full set is computed in `internal/builder/security.go::BuildSecurityHeaders`. The agent should not edit that file; instead, change settings via `bulk_upsert_settings` (general/seo/analytics writable) or, for the security category, direct the user to the admin UI.

## Whitelisting an iframe

When a block embeds a third-party iframe (Cal.com, YouTube, Stripe Checkout), the CSP `frame-src` directive must allow the origin. The flow:

1. Read `atomicsite://site/security_posture` to see current `trusted_domains.frame`
2. If the origin is not present, call `register_allowed_script` (despite the name, the table has a `kind` column that routes per directive: `script`, `frame`, `image`, `media`, `connect`, `all`) with `kind: "frame"`
3. Then `trigger_build`. The builder regenerates `csp_extra_directives` from the table

Do not edit `csp_extra_directives` directly through `bulk_upsert_settings`; that bypasses the per-origin tracking the admin UI needs.

## Scripts with SRI

Third-party scripts that ship with `integrity="sha384-..."` get auto-emitted with the SRI attribute. When you call `register_allowed_script` for a script, include the integrity hash if you have one. The agent can compute it from the script body or quote from the vendor's documentation.

## Why raw_astro is admin-only

`raw_astro` accepts arbitrary Astro and is the only block that can ship inline `<script>` content. Inline scripts violate the default CSP unless the `'unsafe-inline'` directive is added, which destroys the protection. The block's admin-only flag exists so a compromised agent key cannot mint inline JS that runs in tenant context.

If you need a small script for a specific page, prefer:

- A registered allowed script with the body fetched from the operator's static host
- An iframe to a sandboxed origin
- A block-scoped `<style>` (Astro auto-extracts and the CSP allows `style-src 'self' 'unsafe-inline'` for the extracted form)

## What you must not do

- Do not propose disabling HSTS or HSTS preload to make a local-dev iframe work
- Do not paste `<script src="https://cdn.example.com/...">` into a page body; route through `allowed_scripts`
- Do not edit `frame_ancestors` to allow all origins; if a clickjacking-protected embed needs to be framed, add the specific parent origin

## Where eval guards this

`internal/eval/security.go` checks: header presence, HSTS max-age >= 1 year for preload eligibility, no `'unsafe-inline'` in `script-src`, no `'unsafe-eval'`, frame ancestors list is finite, all third-party scripts have SRI. The agent reads `atomicsite://eval/latest` after a build and remediates per-finding rather than blanket-relaxing the policy.

## CRLF injection guard

`internal/builder/security.go::BuildSecurityHeaders` strips control chars from every settings value before it lands in a header line. The agent does not need to sanitise; the builder does. But you should never propose a settings value containing a literal newline or carriage return; the validator will reject it.
