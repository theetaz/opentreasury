# OpenTreasury Design Language

| | |
|---|---|
| **Status** | v1 — adopted 2026-07-09 |
| **Applies to** | `apps/web` (operator dashboards + public portal), `apps/mobile`, docs sites |
| **Canonical tokens** | [`tokens.css`](tokens.css) — the only source of color/shape/layout values |
| **Live preview** | [`preview.html`](preview.html) — open in a browser, toggle light/dark |
| **Influences** | Uxcel dashboard (card grid, stat groupings), Cloudflare dashboard (data density, tables + sparklines), Modal / Whop / Vapi (dark developer-tool aesthetic) |

---

## 1. Principles

1. **Numbers are the interface.** Every design decision defers to the legibility
   of financial figures: tabular numerals, right-aligned amount columns, high
   ink contrast, no decoration competing with data.
2. **Dense, but hierarchical.** Operators scan dozens of figures per screen
   (Cloudflare-style density); hierarchy comes from type scale, spacing, and
   hairline separation — never from loud color.
3. **Dark-first for operators, light-first for the public.** Treasury and
   institution dashboards default to dark (with a toggle); the public portal
   defaults to the visitor's system preference. Both themes are first-class:
   every token has a selected value per theme, never an automatic inversion.
4. **Color is meaning.** Teal = interactive/brand. Status colors are reserved
   for state and never appear as chart series. Chart series colors follow the
   entity, never its rank, and their order is fixed (§6).
5. **Trust is visible.** Verification affordances (anchor status, proof links,
   "data as of" freshness) are part of the visual language, not footnotes.
6. **Accessible by default.** WCAG 2.1 AA contrast, visible focus rings,
   keyboard-first components, color never the sole carrier of meaning.

## 2. Foundations

### 2.1 Color system

All values live in [`tokens.css`](tokens.css). Roles, not raw hex, everywhere.

| Role group | Tokens | Notes |
|---|---|---|
| Surfaces | `--background`, `--surface`, `--surface-raised`, `--surface-sunken` | Page plane sits slightly off the card surface in both themes; elevation via shadow in light, via lighter surface in dark |
| Ink | `--foreground`, `--muted-foreground`, `--faint-foreground` | Three tiers only — if you need a fourth, the layout is wrong |
| Brand | `--primary`, `--primary-tint`, `--ring` | Teal. Actions, links, active navigation, focus |
| Status | `--success/--warning/--danger/--info` + `-tint` | Reserved. Always paired with an icon or label, never color alone |
| Charts | `--chart-1 … --chart-7`, `--chart-grid`, `--chart-axis` | Fixed-order categorical slots; see §6 |
| Borders | `--border`, `--border-strong` | Hairlines everywhere; strong only for focus/dividers that must read |

Dark is **not** a filter over light: every dark value was selected and the
chart palette re-validated against the dark surface.

### 2.2 Typography

| Use | Face | Size / weight |
|---|---|---|
| UI text | Inter | 14px/400 base; 13px in dense tables |
| Page title | Inter | 24px/600 |
| Section / card title | Inter | 16px/600 |
| Stat value | Inter | 28–32px/650, proportional figures |
| Amount columns, IDs, timestamps | Inter + `tabular-nums` | Right-aligned; `.figures-tabular` |
| Code, hashes, proofs | JetBrains Mono | 13px |

Rules: no serif or display faces anywhere; `tabular-nums` is **mandatory** for
any vertical run of numbers (tables, axis ticks) and omitted for standalone
hero figures; currency amounts render from minor units with exactly two
decimals and an explicit ISO code (`USD 1,250.00`), never a bare symbol.

### 2.3 Spacing, shape, elevation

- 4px base grid; component padding steps 8 / 12 / 16 / 24; card padding 20–24px.
- Radius: cards 12px, controls 8px, pills/badges full.
- Borders are hairlines (`--border`); shadows are whispers (`--shadow-card`).
  Dark mode leans on surface steps more than shadows.
- Density tiers: `comfortable` (default), `dense` (tables -2px font, -4px row
  padding). No "spacious" tier — this is an operator tool.

### 2.4 Iconography

Lucide icons (ships with shadcn/ui), 16px in controls and nav, 20px in empty
states, 1.5px stroke. Icons never appear without a label except in the
collapsed sidebar (where the label becomes a tooltip).

## 3. Layout system

### 3.1 App shell (operator dashboards)

```
┌─────────┬──────────────────────────────────────────────┐
│         │ Topbar 56px: breadcrumb · search ⌘K · env    │
│ Sidebar │  badge · theme toggle · user                 │
│ 248px   ├──────────────────────────────────────────────┤
│ (64px   │ Content: max 1440px, 24px gutters,           │
│ rail    │ 12-col CSS grid, 16px gap                    │
│ when    │                                              │
│ folded) │                                              │
└─────────┴──────────────────────────────────────────────┘
```

- Sidebar matches the theme (no forced-dark sidebar in light mode). Sections:
  product areas, then MANAGE, then HELP (Uxcel pattern). Active item = teal
  tint pill (`--primary-tint`) + teal label; hover = neutral wash.
- Topbar always carries: environment badge (`LOCAL` / `STAGING` / `PROD`),
  API health dot, and the **data freshness stamp** ("as of 14:02, anchored
  14:05") — trust affordances live in the chrome, not in each card.
- Public portal uses a top-nav shell instead (no sidebar), same tokens.

### 3.2 Page templates

| Template | Grid | Used by |
|---|---|---|
| **Dashboard** | Stat-tile row (4×3col) → chart row (2×6col) → table row (12col) | Overview, treasury, institution |
| **Data browser** | Filter bar → full-width table + pagination footer | Transactions, audit trail |
| **Detail** | 8col main + 4col meta rail | Entry detail, institution detail |
| **Workbench** | 6col form + 6col live result | Validation |
| **Explorer (public)** | Search hero → result cards → proof panel | Public portal |

## 4. Components (shadcn/ui base)

Component primitives come from shadcn/ui, themed by the tokens; these are the
OpenTreasury-specific rules on top:

- **Stat tile** — label (13px muted, with info tooltip), value (28px,
  proportional), delta chip (`--delta-up/-down` + arrow, always with sign),
  optional sparkline. Never more than 4 per row.
- **Card** — title row (16px/600 + "View all" text-link right-aligned),
  optional 13px muted timeframe subtitle ("Last 30 days"), body. One concern
  per card.
- **Table** — sticky header on `--surface-sunken`, 13px header labels, hairline
  row separators only (no zebra), amount columns right-aligned tabular, row
  hover wash, entity IDs in mono with copy-on-click. Empty state = icon +
  one-line explanation + primary action (Uxcel pattern).
- **Status badge** — tint background + status ink + label, pill radius:
  `ACTIVE`, `POSTED`, `ANCHORED`, `QUARANTINED`, `REVERSED`… Semantic colors
  only; unknown states render neutral.
- **Verification chip** — the trust affordance: shield icon + "Anchored" +
  block link; clicking opens the proof panel. Teal when verified, warning when
  pending, danger when verification fails.
- **Forms** — labels above inputs, 13px; helper/error text 12px below;
  validation errors inline next to the field AND summarized at top for long
  forms. Monospace inputs for IDs/amounts.
- **Buttons** — primary (teal solid), secondary (outline), ghost (nav/table
  actions), destructive (danger solid, always with confirm). One primary per
  view region.
- **Toasts** — bottom-right, status-tinted border-left, auto-dismiss except
  danger.

## 5. Interaction & motion

- Transitions 150ms ease-out (hover), 200ms (popovers), 250ms (drawers/theme
  cross-fade). No parallax, no spring physics — this is an instrument panel.
- Every interactive element: visible focus ring (`--ring`, 2px offset).
- Keyboard: ⌘K command palette (search + navigation), `[` collapses sidebar,
  table rows focusable with Enter-to-open.
- Skeletons (not spinners) for loading regions; spinner only inside buttons.
- Live regions (`aria-live=polite`) announce async results (validation,
  refresh).

## 6. Data visualization

The chart system follows the repo-agnostic method: form first, color by job,
**validated palette**, thin marks, hover layer, accessibility pass.

### 6.1 Categorical palette (fixed order — never cycle, never reorder)

| Slot | Hue | Light | Dark | Note |
|---|---|---|---|---|
| 1 | teal | `#0d9488` | `#0da895` | brand |
| 2 | blue | `#2a78d6` | `#3987e5` | |
| 3 | amber | `#eda100` | `#c98500` | sub-3:1 on light → relief rule |
| 4 | violet | `#4a3aa7` | `#9085e9` | |
| 5 | green | `#008300` | `#279f43` | |
| 6 | magenta | `#e87ba4` | `#d55181` | sub-3:1 on light → relief rule |
| 7 | orange | `#eb6834` | `#d95926` | |

Validation results (2026-07-09, `validate_palette.js`): **light** on `#ffffff`
— all checks pass, worst adjacent CVD ΔE 13.8; **dark** on `#131316` — all
checks pass incl. every slot ≥3:1, worst adjacent ΔE 13.4. Amber and magenta
are below 3:1 contrast on light: any chart using slots 3/6 in light mode ships
visible direct labels or a table view (the "relief rule"). An 8th series is
never a new hue — fold into "Other" or use small multiples.

### 6.2 Rules of the house

- **One axis.** Never dual-axis; two measures → two charts or indexed series.
- Sequential = teal ramp light→dark; diverging = teal ↔ orange with neutral
  gray midpoint (inflow/outflow); status palette never doubles as series.
- Marks: 2px lines, 4px rounded ends on bars (anchored to baseline), 2px
  surface gaps between stacked/adjacent fills, ≥8px hover markers.
- Grid: hairline `--chart-grid`, horizontal only where possible; baseline
  `--chart-axis`; axis text `--faint-foreground` 12px tabular.
- Hover layer by default: crosshair + tooltip on time series, per-mark tooltip
  elsewhere. Tooltip = raised surface, 12px, entity marker + label + value.
- Legend whenever ≥2 series; ≤4 series also direct-labeled; single series
  needs no legend (the title names it).
- Money on axes: compact notation (`1.2M`), full precision in tooltips.

## 7. Theming implementation

- Tokens are CSS custom properties (`tokens.css`); Tailwind reads them via
  `hsl(var(--…))`-style mapping in the Phase 5.1 web revamp; shadcn components
  inherit automatically.
- Theme = `.dark` class on `<html>`, set by: stored user choice → else system
  preference. Operator app default = dark; public portal default = system.
  Toggle lives in the topbar; switch cross-fades 250ms.
- Charts re-read tokens on theme switch (CSS vars make this free); chart
  palettes are theme-specific values of the same slot tokens.

## 8. Voice & content

- Sentence case everywhere (buttons, titles, labels). No ALL-CAPS except tiny
  section eyebrows (`MANAGE`) and status badges.
- Empty states explain *why* and give one action ("No course activity in the
  last 30 days — adjust the timeframe or assign a course").
- Errors: what failed + what to do, no codes without words, request ID
  available behind a "details" disclosure.
- Timestamps: absolute with timezone in tables (`2026-07-09 14:02 GMT+2`),
  relative in activity feeds ("4 min ago"), always machine-readable `<time>`.

## 9. Adoption plan

1. **Phase 5.1 (pulled forward):** rebuild `apps/web` on this system —
   Tailwind + shadcn/ui + tokens.css, React Router shell (sidebar/topbar),
   Zustand + TanStack Query, dashboard/data-browser/workbench templates.
2. `packages/ui` becomes the home of tokens + shared primitives; this doc and
   `tokens.css` move contributors' questions from taste to reference.
3. Public portal and mobile consume the same tokens when they land (P4.5/P6.7).
4. Any token change requires: palette re-validation for `--chart-*`, AA
   contrast check for ink/status, and a PR referencing this doc's section.
