# Dark mode

Dark mode is its own composition, not light mode with a hue rotation. The agent's job is to design the dark variant intentionally and let the variable system carry the rest.

## How Atomic Site implements it

The admin SPA toggles via `[data-theme="dark"]` on `<html>`. Built sites adopt the same pattern when dark mode is enabled in the branding admin. Every CSS variable has a light-mode value and a dark-mode value:

```css
:root {
  --color-bg      : #ffffff;
  --color-surface : #f8f9fb;
  --color-border  : #e5e7eb;
  --color-muted   : #6b7280;
  --color-text    : #111827;
}

[data-theme="dark"] {
  --color-bg      : #0b0d12;
  --color-surface : #14171f;
  --color-border  : #2a2f3a;
  --color-muted   : #94a3b8;
  --color-text    : #e5e7eb;
}
```

Blocks reference variables only; they do not need a `[data-theme="dark"]` selector of their own. The cascade does the work.

## Dark backgrounds are not black

`#000` reads as a hole, not a surface. Use a deep cool gray (`#0b0d12`, `#101218`) for backgrounds, slightly lifted gray (`#14171f`) for surfaces, brighter gray (`#2a2f3a`) for borders. The contrast feels deliberate; pure black feels like the screen is broken.

## Dark text is not pure white

`#ffffff` on a dark background is too high-contrast and feels harsh during long reads. Use `#e5e7eb` (a near-white at ~92% luminance). It still passes WCAG AA against the dark backgrounds above and reads as professional rather than fluorescent.

## Brand colors in dark mode

The operator's primary color often needs adjustment for dark mode. A saturated brand red on white reads as warm; on a dark background it can read as alarming or muddy. Two patterns:

- **Slightly desaturate**: brand `#dc2626` -> dark `#ef4444` (less saturation, more luminance)
- **Slightly lift**: brand `#1d4ed8` -> dark `#3b82f6` (lift hue toward sky)

The branding admin can store both light and dark values for `--color-primary`. When it does not, the agent derives the dark value with `color-mix()` against the dark surface.

## Shadow strategy

Light mode shadows imply elevation via a soft drop. Dark mode shadows almost never work; the dark surfaces already swallow them. Replace shadows with subtle borders or surface lifts:

```css
.card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  /* no box-shadow in dark mode; --color-surface > --color-bg is the elevation */
}
```

## Focus rings

The focus ring color must work against both light and dark surfaces. A high-saturation primary works against light; against dark you may need a brighter variant. The default pattern uses `--color-primary` for both and accepts the slight asymmetry; tune via the branding admin if the contrast falls below 3:1.

## Toggle UX

When the site exposes a theme toggle, the toggle:

- Is in the header, not buried in a footer link
- Has three states: light, dark, system (matches OS preference)
- Persists in `localStorage` keyed by the site domain
- Defaults to system on first visit
- Hydrates without flash: a small inline `<script>` in `<head>` reads `localStorage` and sets `[data-theme]` before paint

The flash-of-wrong-theme on first paint is one of the most visible UX failures. Fix it with the inline script.

## Images and media in dark mode

Logos with white backgrounds read awful on dark surfaces. The branding admin can store a dark-variant logo; the builder picks the right one based on `[data-theme]`. When only a light logo exists, wrap it in a light-tinted container at small sizes rather than punching a white card into the dark layout.

For inline images in content, leave them alone. Photographs and illustrations should not change between modes.

## What you must not do

- Do not invert colors with a CSS filter; the result is uniformly bad
- Do not introduce dark-mode-only block variants if the variable system can carry the change
- Do not use the same primary as accent and link color in dark mode without testing contrast against three surfaces (background, surface, elevated)
- Do not hide dark mode behind a settings page; if you support it, expose the toggle

## What the eval grades

The accessibility eval runs contrast checks against the active theme. When dark mode is enabled, dark variant contrasts must independently meet AA. The eval does not check for "is dark mode enabled"; that is an operator choice. But once enabled, both themes must pass.
