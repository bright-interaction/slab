# Accessibility patterns

Atomic Site's accessibility eval (`internal/eval/accessibility.go`) checks WCAG 2.1 AA conformance plus a few atomicsite-specific rules. The agent's job is to write blocks that pass the eval out of the box.

## Landmarks

Every page needs four landmarks: `<header>`, `<nav>`, `<main>`, `<footer>`. The default `Layout.astro` provides them. The agent should never replace them with `<div>` wrappers when adding global blocks; the global header block emits `<header>` semantics, the global footer emits `<footer>`, and so on.

## Headings

One `<h1>` per page. Subsequent headings descend in order: `<h2>`, then `<h3>`, never skipping. The hero block owns the h1; no other block emits one. The eval flags multiple h1s and skipped levels.

## focus-visible (the modern pattern)

Atomic Site's `internal/builder/css.go:432` emits this exact pattern, which the agent must preserve in custom CSS:

```css
button:focus,
a:focus,
input:focus,
textarea:focus,
select:focus {
  box-shadow: 0 0 0 2px var(--color-bg), 0 0 0 4px var(--color-primary);
}
button:focus:not(:focus-visible),
a:focus:not(:focus-visible) {
  outline: none;
  box-shadow: none;
}
button:focus-visible,
a:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

The pattern: mouse clicks get no visible focus ring (clean look). Keyboard navigation gets a high-contrast outline (WCAG 2.1 SC 2.4.7 fail without it). Do not regress to `outline: none` everywhere.

## Skip-to-content link

The first focusable element on every page is a skip link that jumps to `<main>`:

```astro
<a class="skip-link" href="#main">Skip to content</a>
```

```css
.skip-link {
  position: absolute;
  inset-block-start: -100px;
  inset-inline-start: 0;
  padding: var(--space-2) var(--space-4);
  background: var(--color-primary);
  color: var(--color-bg);
  z-index: 100;
}
.skip-link:focus {
  inset-block-start: 0;
}
```

The default layout includes this. Custom global headers must not omit it.

## Form labels

Every `<input>`, `<textarea>`, `<select>` has a corresponding `<label>` (or `aria-label`). The eval flags forms without labels. Placeholder text is not a substitute for a label; placeholders disappear on focus and screen readers cannot use them as field names.

```astro
<label for="email">Email</label>
<input id="email" name="email" type="email" autocomplete="email" required>
```

## autocomplete attributes

Email, name, tel, address, password fields all need `autocomplete` attributes. The accessibility eval and the form-ux eval both check this. The values follow the HTML autofill spec: `email`, `name`, `tel`, `street-address`, `current-password`, `new-password`.

## tabindex

Allowed values: `0` (focusable in document order), `-1` (programmatically focusable but not in tab order). Never positive (`tabindex="3"`). Positive values reorder the tab sequence and almost always break keyboard navigation downstream.

## aria-label for JS-injected anchor labels

When a block emits an anchor whose visible text comes from JavaScript (e.g., a stat counter that animates from 0 to the target), the anchor must have an `aria-label` so screen readers have a stable name. The animated text alone leaves screen readers reading "0" until the animation completes.

## Color contrast

Body text contrast ratio against background must meet 4.5:1 (WCAG AA). Large text (18.66px+ regular or 14px+ bold) meets 3:1. UI components (button borders, focus rings, form outlines) meet 3:1. The eval computes contrast for every color pair the page emits.

When a brand color fails contrast, do not silently lower the requirement. Either:
- Use the color for non-text accents only
- Derive a darker variant for text-on-background uses
- Switch to a foreground neutral and use the brand as a border

See `atomicsite://knowledge/color-system` for the derivation pattern.

## Motion

Respect `prefers-reduced-motion`. See `atomicsite://knowledge/motion-curves`.

## Touch targets

Interactive elements (buttons, links in clusters, form controls) must be at least 44x44px on mobile. Inline links inside paragraphs are exempt; standalone buttons are not. The default block CSS sizes buttons to `padding: var(--space-3) var(--space-6)` which clears the threshold.

## ARIA and roles

Use semantic HTML before reaching for ARIA. `<button>` not `<div role="button">`; `<nav>` not `<div role="navigation">`. ARIA is a fallback for cases the HTML primitives do not cover (custom comboboxes, tree views, live regions). When the HTML is right, ARIA is mostly unnecessary.
