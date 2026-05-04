# Motion curves

Motion is one of the strongest signals of polish. It is also one of the easiest places to make a site feel cheap. The discipline is short list, predictable timing, hardware-accelerated transforms only, and a `prefers-reduced-motion` opt-out.

## Timing tokens

```
--motion-fast    : 150ms   /* hover, focus state changes */
--motion-normal  : 200ms   /* default transition duration */
--motion-slow    : 300ms   /* enter / exit, modal show */
--motion-deliberate: 500ms /* hero reveal, page load */
```

Anything above 500ms feels sluggish for UI. Anything below 100ms is below perception and feels broken (the user does not perceive feedback). Stay inside this range.

## Easing curves

Three curves cover almost everything:

```
--ease-out  : cubic-bezier(0.2, 0.8, 0.2, 1)   /* default; decelerate into place */
--ease-in   : cubic-bezier(0.4, 0, 1, 1)        /* accelerate away */
--ease-soft : cubic-bezier(0.4, 0, 0.2, 1)      /* material-style soft */
```

Use `--ease-out` for most things: hover feedback, content reveals, modal opens. Use `--ease-in` for things leaving (modal close, dismissed toast). `--ease-soft` is a workhorse when the others feel wrong; it rarely dominates a design.

Avoid `linear` for anything except progress indicators. Linear motion feels mechanical.

## Hardware-accelerated only

Animate only `transform` and `opacity`. Both compose on the GPU and avoid layout reflow. Do not animate `width`, `height`, `top`, `left`, `padding`, or `margin`; they trigger layout and stutter on low-end devices.

```css
/* Good */
.card {
  transition: transform var(--motion-normal) var(--ease-out),
              opacity var(--motion-fast) var(--ease-out);
}
.card:hover {
  transform: translateY(-2px);
}

/* Bad */
.card {
  transition: top var(--motion-normal) var(--ease-out);
}
.card:hover {
  top: -2px;
}
```

## prefers-reduced-motion

Wrap any motion that crosses the threshold of "pleasant" (page-load reveals, autoplay carousels, parallax) in:

```css
@media (prefers-reduced-motion: no-preference) {
  .reveal {
    animation: fadeUp var(--motion-deliberate) var(--ease-out);
  }
}
```

Hover and focus state transitions stay; they are functional feedback, not decoration. The accessibility eval does not require disabling them.

## Micro-motion

Premium sites have perpetual subtle motion: a button shifts 1px on hover, an icon nudges on focus, a card lifts 2px. The motion is small, fast, predictable. The agent's job is to add this discipline to every interactive element, not to add big animations.

```css
.btn {
  transition: transform var(--motion-fast) var(--ease-out),
              background var(--motion-fast) var(--ease-out);
}
.btn:hover {
  transform: translateY(-1px);
}
.btn:active {
  transform: translateY(0);
}
```

## What you must not do

- Do not autoplay video or carousels longer than 5 seconds without an explicit user opt-in
- Do not animate scroll-jacking (overriding the browser's scroll). It breaks accessibility and usually feels gimmicky
- Do not stack three animations on the same element with sequential delays. One element, one motion.
- Do not use motion to disguise slow loads. Fix the load.

## What the eval grades

Eval does not directly score motion, but heavy CSS animations that animate non-composited properties affect LCP and CLS. The CLS score (cumulative layout shift) penalises anything that reflows on first paint. Stick to transform + opacity and CLS stays clean.
