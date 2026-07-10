# CookieProof integration

CookieProof is the consent banner Atomic Site embeds on every published site. It implements the IMY 2026 equal-prominence rule (Reject and Accept share the same primary color and visual weight), exposes a per-category toggle, and cleans up pre-consent tracker cookies that leak from parent domains.

## Default behaviour

When `analytics.cookieproof_enabled` is on (default 1 since 2026-05-01), every page emits the embedded widget bundle. No external request, no third-party CDN; the widget script is `go:embed`-baked into the binary and served at `/_ccb.<hash>.js`.

## Equal prominence

The IMY 2026 rule says Reject must be as visually prominent as Accept. Atomic Site enforces this in the widget: both buttons render with the site's primary color, the same size, the same weight. The agent must not propose a banner where Reject is "subtle" or text-only; the eval engine flags it.

## Pre-consent cookie sweep

Some parent domains (`.example.com`, customer apex domains) set tracker cookies that subdomains inherit on first visit. The widget walks a default 20-entry list of patterns (`_ga`, `_ga_*`, `ph_*`, `rl_*`, `_fbp`, `_hj*`, etc.) and expires them on init when analytics + marketing are not granted. The list is overridable via `CookieConsentConfig.cleanupTrackingCookies` in custom embeds, but the default covers the common bleed.

## Cookie tables

Each consent category lists the actual cookies that fire when granted. Atomic Site stores them in `analytics.cookie_declarations` as JSON; the widget renders one row per cookie with name, purpose, duration, provider. The agent can edit declarations via `bulk_upsert_settings`; the JSON shape is documented in `atomicsite://site/settings_catalog`.

## Language toggle

When the site has additional languages (`general.additional_langs`), the widget shows a language selector inside the preferences modal. The agent does not need to wire it; the builder reads the additional langs and generates the translation set.

## Privacy policy link

`analytics.cookie_banner_privacy_url` should point at the site's privacy policy (typically `/privacy/`). The widget renders a "Privacy Policy" link in the banner header and the preferences modal. If the URL is empty, the link is hidden; eval flags this as a Privacy gap.

## Custom override surface

Operators can override defaults at every layer:

- Title, description, button labels: `cookie_banner_title`, `cookie_banner_description`, `cookie_banner_accept`, `cookie_banner_reject`, `cookie_banner_customize`
- Position: `cookie_banner_position` enum (bottom-center default, bottom-left, bottom-right, top-center)
- Theme: `cookie_banner_theme` enum (light, dark, auto)
- Inline CSS or HTML: `cookie_banner_snippet` for advanced overrides

The agent uses these for tone matching (a serious legal site might prefer different copy than a casual SaaS). All overrides flow through validation; CSP-relevant fields strip control chars.

## CCPA mode

`analytics.cookie_banner_ccpa` toggles the "Do Not Sell My Personal Information" link required for California visitors. Default is off; turn it on for sites with US visitors. The widget renders the required link and respects the GPC (Global Privacy Control) header.

## What the eval grades

`internal/eval/privacy.go` checks: banner present, equal-prominence buttons, cookie declarations match what's actually set, privacy policy link present, language coverage matches additional langs, plaintext emails obfuscated, GDPR clause in footer.

## Live preview

Settings -> Cookies in the admin shows the real widget mounted in an iframe with the operator's current branding and copy. The agent can direct users there for visual confirmation; the preview path uses `RenderCookieProofConfigPrefix` from the same builder code that ships to production, so what the editor sees is what visitors see.
