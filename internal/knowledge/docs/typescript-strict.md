# TypeScript strict mode for Astro components

The built site uses TypeScript strict mode. Astro components are TypeScript by default; the props interface lives at the top of every `.astro` file. The agent should write idiomatic strict-mode TS, never `any`.

## Prop typing

Every component declares its props with `interface Props` and a frontmatter destructure:

```astro
---
interface Props {
  title: string;
  description?: string;
  ctaLabel: string;
  ctaHref: string;
}

const { title, description, ctaLabel, ctaHref } = Astro.props;
---
```

Optional props use `?:`. Required props that can be empty (an empty string is still a string) stay required. Default values go on the destructure: `const { variant = "primary" } = Astro.props;`.

## Type-only imports

Imports that exist only at the type layer use `import type`:

```ts
import type { Page, Block } from "../types/db";
```

This keeps the runtime bundle clean. Mixing `import` for types makes esbuild emit empty stub modules.

## No `any`

`any` is banned. When you do not know the shape, use `unknown` and narrow:

```ts
const data = JSON.parse(raw) as unknown;
if (typeof data === "object" && data !== null && "id" in data) {
  // narrowed
}
```

For block `data_json` payloads, type the schema as `Record<string, unknown>` and narrow per field. The block registry in `internal/blocks/registry.go` is the source of truth for valid keys; mirror its kinds in your TypeScript shapes.

## Discriminated unions for block variants

When a block has multiple visual modes (e.g., a hero with `centered` vs `split`), model it as a discriminated union:

```ts
type Hero =
  | { layout: "centered"; headline: string; subhead?: string }
  | { layout: "split"; headline: string; image: string; align: "left" | "right" };
```

The render path then narrows on `layout` and the compiler enforces every branch.

## Prefer `const` and arrow functions

`let` is for mutable bindings only. Function expressions for callbacks. Named functions for top-level exports. Match the rest of the codebase rather than inventing a personal style.

## Astro-specific gotchas

- `Astro.props` is typed by the local `Props` interface; do not destructure with type assertions, the inference is enough.
- `Astro.url`, `Astro.params`, `Astro.glob` are typed by Astro; do not redeclare them.
- `class:list` accepts `(string | Record<string, boolean>)[]`. Use it instead of string-concat for conditional classes.
- Slots are typed via `<slot />`; named slots with `<slot name="header" />`. The agent rarely declares slots in dynamically generated blocks; the layout shell already exposes the ones the design system needs.

## Where strict mode bites

`tsconfig.json` enables `strict`, `noUncheckedIndexedAccess`, and `exactOptionalPropertyTypes`. Two things this catches:

- `arr[0]` is `T | undefined`, not `T`. Guard or destructure with a default.
- `{ x?: number }` is not assignable to `{ x: number | undefined }`. Use `?:` or set the value explicitly to `undefined`.

When a build fails on either of these, fix the type rather than weaken the compiler. The strictness is what keeps long-running sites consistent.
