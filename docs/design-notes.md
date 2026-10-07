# Design notes

Concise record of the visual/design system used across the app, so new
screens stay consistent. UI text is English (PRD D-12).

## Tokens

Single source of truth: `frontend/src/styles/tokens.css`. All values are CSS
custom properties consumed directly by components (no runtime theme switch —
dark mode is out of scope, PRD D-13).

| Token                  | Value                         | Used for                             |
| ---------------------- | ----------------------------- | ------------------------------------ |
| `--color-bg`           | `#ffffff`                     | page background                      |
| `--color-surface`      | `#f7f8fa`                     | cards, panels                        |
| `--color-border`       | `#e2e5ea`                     | card borders, dividers               |
| `--color-text`         | `#1a1d21`                     | body text (15.4:1 on bg)             |
| `--color-text-muted`   | `#5c6470`                     | secondary text (6.2:1 on bg)         |
| `--color-accent`       | `#2563eb`                     | primary actions, links (5.2:1 on bg) |
| `--color-accent-hover` | `#1d4ed8`                     | hover state                          |
| `--color-danger`       | `#dc2626`                     | destructive actions                  |
| `--color-danger-hover` | `#b91c1c`                     | destructive hover                    |
| `--radius-sm`          | `6px`                         | inputs, small buttons                |
| `--radius-md`          | `10px`                        | cards                                |
| `--space-1..6`         | 4/8/12/16/24/32px             | all spacing (no magic numbers)       |
| `--font-body`          | system-ui stack               | everything                           |
| `--font-mono`          | ui-monospace stack            | short codes, URLs, stats             |
| `--shadow-card`        | `0 1px 2px rgb(0 0 0 / 0.06)` | cards                                |

## Typography

- Body 16px/1.5; headings 20/24/32px semibold, tightened line-height 1.2.
- Short codes and URLs always `--font-mono`, `word-break: break-all` inside
  copy blocks so long URLs wrap instead of overflowing (§17.3).
- Minimum tap target 44×44px for buttons/links on touch widths.

## Layout

- Content column: `max-width: 640px`, centered, `--space-4` page padding.
  Dashboard detail pages may use `max-width: 960px` for the chart.
- Cards stack vertically below 640px; two-column grid (form 1fr, preview
  auto) only at ≥ 900px.
- Breakpoints: 320 (baseline), 480, 640, 900, 1200. Every layout is verified
  at 320px and 375px via Playwright (§17.3) — `document.scrollingElement`
  must not overflow horizontally.

## Interaction states

- Buttons: default / hover / active (translateY 1px) / focus-visible ring
  (`outline: 2px solid var(--color-accent); offset 2px`) / disabled (50%
  opacity, `cursor: not-allowed`).
- Inputs: 1px border, on focus border-color accent + same visible ring.
  Errors: danger border + text below, linked via `aria-describedby`.
- Loading: buttons show inline spinner and `aria-busy="true"`; lists show a
  skeleton row instead of a spinner to avoid layout shift.

## Motion

- Transitions 150ms ease-out on color/background only (prefers-reduced-motion
  disables them via media query).
- No entrance animations, no parallax, no auto-playing carousels.

## Accessibility rules

- Every interactive element reachable and operable by keyboard; logical tab
  order; no positive `tabindex`.
- Icon-only buttons have `aria-label`.
- Copy-success feedback uses a visually hidden live region
  (`aria-live="polite"`), not color alone.
- Contrast: all text/background pairs meet WCAG AA (tokens above ≥ 4.5:1;
  large text ≥ 3:1).
- Form fields have real `<label>` elements; errors reference inputs.
- Skip link ("Skip to content") as first focusable element on every page.

## Copy tone

Short, imperative, no exclamation marks. Errors state what happened and the
next step ("That alias is taken. Try another one."). Buttons are verbs
("Create link", "Copy", "Delete").
