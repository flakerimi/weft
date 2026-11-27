# Landing blocks: `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta`

Date: 2026-11-27
Extends `2026-11-20-weft-design.md` (Components tier 5, phase 7) and `docs/roadmap.md` (`0.1.0`).

## Intent

A Weft product starts on a landing page (`surface="landing"`) before a visitor signs in to the app (`surface="app"`). `@weft/ui` already ships the shell (`Header`, `Footer`), the layout primitives (`Page`, `Section`, `Container`, `Grid`, `Stack`, `Card`, `Divider`, `Separator`) and the interactive controls (`Button`, `Badge`, `Avatar`, `Accordion`, `Switch`, `ToggleGroup`).

The six landing blocks (`Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta`) are the high-level page sections an author or coding agent assembles a product's marketing or documentation home page from—without writing custom flex/grid markup or bespoke CSS per page, while keeping full slot escape hatches when a block holds a live preview, code panel, or illustration (as weft.base.al's own landing page does).

## Goals

1. Six Vapor components in `packages/ui/src/landing/`: `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta`.
2. **One prop vocabulary with the rest of `@weft/ui`:**
   - Actions use `ActionItem` (`{ label, onClick, to, icon, tone, variant, disabled, loading }`, from `../feedback/actions.ts`), rendered with `Button`, with an `actions` slot to replace them.
   - Side-by-side vs stacked layout uses `orientation: 'vertical' | 'horizontal'` everywhere (`Hero`, `Faq`, `Cta`), matching `Tabs`, `RadioGroup`, `ToggleGroup`, `Stepper` and `Separator`. `align: 'start' | 'center'` is used only for text and action alignment, matching `Stack`.
   - Heading levels are configurable (`level?: 1 | 2 | 3 | 4 | 5 | 6` on `Hero`, `level?: 2 | 3 | 4 | 5 | 6` on the section blocks). `Hero` defaults to `h1`, or `h2` inside an `inset` `WeftProvider` (`useInset()`) like `Page` does, so the gallery stage never adds a second `h1` to the page; the section blocks default to `h2` and render child headings one level deeper (`Math.min(level + 1, 6)`).
   - Layout and columns adapt from the block's **own container width** via `useWidth(root)` (`narrow` under 640 px, `regular` under 1024 px, `wide` 1024 px and up), never `window.innerWidth`, so a block inside a split preview or gallery stage adapts to the box it is given.
   - Touch adaptation follows `@weft/ui`'s rules: buttons, links, billing toggles and FAQ triggers are at least `--weft-hit-min` (44 px on touch) tall.
3. **Zero JavaScript by default on static pages:** `Hero`, `FeatureGrid`, `Pricing` (without a billing cycle switch), `Testimonials`, and `Cta` are pure server-rendered HTML when given no interactive listeners, and are added to the gallery's `STILL` set. `Faq` wraps `Accordion` (`hidden="until-found"`, WAI-ARIA APG Accordion keyboard navigation, find-in-page support), and `Pricing` with `cycles` uses `ToggleGroup`.
4. **Tokens only:** every stylesheet in `packages/ui/src/styles/` reads only `--weft-*` custom properties with system fallbacks, imported per component and grouped in `packages/ui/src/styles/landing.css` (which `weft.css` imports). No new theme tokens are required; existing colour, surface, spacing, radius, border, shadow, font-weight and motion tokens cover all six blocks in light and dark schemes across `neutral`, `lisa` and `weft`.

## Non-goals

- `Carousel` and `Marquee` (auto-scrolling logo strips): motion-heavy widgets not needed for `0.1.0`'s six landing blocks.
- Site-specific decorative artwork (weft.base.al's woven background mask and animated saffron thread stay in `site-src/`; `Hero` provides a `background` slot and `default` media slot so a page can place any illustration or preview frame inside it).

---

## Visual language & token use

From `site-src/styles/landing.css` and `packages/themes`:

- **Typography:**
  - Eyebrow / headline above a title: `--weft-color-primary-text` (or a `Badge` when slotted), `font-weight: var(--weft-weight-heading, 600)`, `font-size: 0.875em`.
  - `Hero` title (`h1`, or `h2` when `useInset()` is true): `font-size: clamp(2rem, 4.2vw, 3.5rem)`, `font-weight: var(--weft-weight-title, 700)`, `line-height: 1.05`, `letter-spacing: -0.035em`, `text-wrap: balance`.
  - Section block titles (`h2` by default): `font-size: clamp(1.5rem, 3vw, 2.25rem)`, `font-weight: var(--weft-weight-title, 700)`, `line-height: 1.1`, `letter-spacing: -0.025em`, `text-wrap: balance`.
  - Lead / description copy: `--weft-color-text-muted`, constrained to a readable measure (`max-inline-size: 60ch`; `46ch` in horizontal headers).
- **Surfaces & cards:**
  - Feature cards, pricing plans, and testimonial cards sit on `--weft-color-surface` (or `--weft-color-surface-raised` for a highlighted pricing plan / `Cta` card), with `border: var(--weft-border-width, 1px) solid var(--weft-color-border, GrayText)`, `border-radius: var(--weft-radius-card, 12px)` (or `--weft-radius-panel, 16px` on `Cta` and highlighted plans), `box-shadow: var(--weft-shadow-card, none)`.
  - A highlighted `Pricing` plan (`highlight: true`) edges its border in `--weft-tone` (`--weft-border-width-strong, 2px`) and shows its `badge` (e.g. `"Most popular"`).

---

## Component designs

### 1. `Hero` (`packages/ui/src/landing/Hero.vue`, `styles/hero.css`)

**Purpose:** The opening band of a landing page: a headline, lead copy, primary and secondary calls to action, and an optional visual (an app preview, image or code block) below or beside the copy.

**Props:**
- `headline?: string` — short eyebrow text above the title (or use the `headline` slot, e.g. for a `Badge` or announcement link).
- `title?: string` — main heading. Rendered as `h1` by default, or `h2` when inside an `inset` `WeftProvider` (`useInset()`), or `h${level}` when `level` is explicitly set.
- `description?: string` — lead paragraph under the title.
- `actions?: ActionItem[]` — buttons under the description; the first defaults to `variant="solid" tone="primary" size="lg"`, the rest to `variant="outline" tone="neutral" size="lg"`.
- `orientation?: 'vertical' | 'horizontal'` — `'vertical'` (the fixed default: copy above, media below) or `'horizontal'` (copy at the inline start, media in the default slot at the inline end when `wide`; stacks vertically when `narrow` or `regular`).
- `align?: 'start' | 'center'` — text and action alignment (`'center'` by default when `orientation="vertical"`, `'start'` when `orientation="horizontal"`).
- `level?: 1 | 2 | 3 | 4 | 5 | 6` — overrides the heading level.

**Slots:**
- `background?: () => unknown` — full-bleed decorative layer behind the container.
- `headline?: () => unknown` — above the title, in place of `headline`.
- `title?: () => unknown` — inside the heading, in place of `title`.
- `description?: () => unknown` — in place of `description`.
- `actions?: () => unknown` — in place of `actions` (e.g. an install command or prompt box).
- `caption?: () => unknown` — a quiet note below the actions (such as `"Already installed? weft update"`).
- `default?: () => unknown` — the visual (app frames, screenshot, code block, video) under the copy (`vertical`) or beside it (`horizontal`).

**Adaptation:**
- `input`: action buttons use `size="lg"` (44 px high on pointer and touch alike, reaching `--weft-hit-min`).
- `surface`: on `landing` the base text is `1.125rem` and the inner container caps at `--weft-page-max` (`1200px`); in `app` it fills the available width.
- `width`: measured on the `<section class="weft-hero">` root via `useWidth(root)` (`data-width`). When `narrow` or `regular`, a `horizontal` hero stacks its copy above the media slot and wraps action buttons.

---

### 2. `FeatureGrid` (`packages/ui/src/landing/FeatureGrid.vue`, `styles/feature-grid.css`)

**Purpose:** A titled section of product capabilities or benefits, laid out in a responsive multi-column grid of items with icons, titles, descriptions and optional links.

**Types (`packages/ui/src/landing/types.ts`):**
```ts
export interface FeatureItem {
  /** The feature's heading. */
  title: string
  /** A sentence or two explaining the feature. */
  description?: string
  /** A Lucide icon name drawn in a tinted disc above or beside the title. */
  icon?: string
  /** An optional address: makes the feature's link or action point there. */
  to?: string
  /** Text for the item's link when `to` is set (default: the title itself links when no linkLabel is given). */
  linkLabel?: string
  /** Optional slot name to render custom body/media for this tile. */
  slot?: string
}
```

**Props:**
- `headline?: string` — eyebrow text above the section title.
- `title?: string` — section heading (`h2` by default, or `h${level}`).
- `description?: string` — section lead under the title.
- `items?: FeatureItem[]` — the features to render.
- `cols?: number | { narrow?: number; regular?: number; wide?: number }` — columns per width class, passed to `Grid` (default `{ narrow: 1, regular: 2, wide: 3 }`).
- `variant?: 'card' | 'plain'` — `'card'` (default: each item in a bordered surface box) or `'plain'` (unboxed items with icon discs, for denser rows of facts).
- `tone?: Tone` — tone of the icon discs (`'primary'` by default).
- `align?: 'start' | 'center'` — alignment of the section header (`'start'` by default).
- `level?: 2 | 3 | 4 | 5 | 6` — section heading level (default `2`; item titles render one level deeper, `Math.min(level + 1, 6)`).

**Slots:**
- `headline?: () => unknown`
- `header?: () => unknown` — replaces the section heading block.
- `actions?: () => unknown` — section-level actions in the header.
- `item?: (props: { item: FeatureItem; index: number }) => unknown` — custom rendering for an item's content.
- `[name: string]: ((props: { item: FeatureItem; index: number }) => unknown) | undefined` — named slot matching `item.slot` (for bento tiles or live component stages).
- `default?: () => unknown` — additional cards or content inside the grid.

**Adaptation:**
- `input`: any item link with `linkLabel` renders as a `Link` with `standalone`, getting a 44 px touch hit area.
- `surface`: text scale and container max-width follow the surface.
- `width`: `Grid` measures its own box via `useWidth`, reflowing from 1 column (`narrow`) to 2 (`regular`) to 3 (`wide`).

---

### 3. `Pricing` (`packages/ui/src/landing/Pricing.vue`, `styles/pricing.css`)

**Purpose:** A comparison of plans side by side, each with its name, price, billing period, included features, and call-to-action button, with an optional billing-cycle switch (such as monthly vs yearly) built on `ToggleGroup`.

**Types (`packages/ui/src/landing/types.ts`):**
```ts
export interface PricingCycle {
  /** The cycle's id in `v-model:cycle` (e.g. `'monthly'`, `'yearly'`). */
  value: string
  /** Shown on the switch button (e.g. `'Monthly'`, `'Yearly'`). */
  label: string
  /** Optional badge on the cycle button (e.g. `'Save 20%'`). */
  badge?: string
}

export interface PricingPlan {
  /** Plan identifier (default: its title). */
  id?: string
  /** Plan name: `'Starter'`, `'Pro'`, `'Enterprise'`. */
  title: string
  /** Who the plan is for. */
  description?: string
  /** Price text as shown (`'$29'`, `'Free'`, `'Custom'`), or a map by cycle value (`{ monthly: '$29', yearly: '$24' }`). When a map has no entry for the active cycle, its price area renders empty (never showing another cycle's price) and warns once in development. */
  price: string | Record<string, string>
  /** Text after the price (`'/ month'`, `'per seat / month'`), or a map by cycle value (missing cycle shows nothing and warns once in development). */
  period?: string | Record<string, string>
  /** Optional note under the price (`'Billed yearly ($288)'`), or a map by cycle value (missing cycle shows no note). */
  billingNote?: string | Record<string, string>
  /** Optional badge on the card (`'Most popular'`). */
  badge?: string
  /** Emphasise this plan with a strong border in `tone` and a solid primary button. */
  highlight?: boolean
  /** Included features listed with a check icon. */
  features?: string[]
  /** Call-to-action button for the plan. */
  action?: ActionItem
  /** Optional slot name for custom plan content. */
  slot?: string
}
```

**Props:**
- `headline?: string`
- `title?: string`
- `description?: string`
- `plans?: PricingPlan[]` — the plans to compare.
- `cycles?: PricingCycle[]` — when given (e.g. Monthly / Yearly), renders a `ToggleGroup` in the header to pick the active billing cycle, using `ToggleGroup`'s default slot to render a cycle's `badge` (`Badge` with `size="sm"`) beside its `label`. Pressing the active cycle keeps it selected (a billing period is always active).
- `cycle?: string` — the selected cycle value (`v-model:cycle`); defaults to `defaultCycle ?? cycles?.[0]?.value`.
- `defaultCycle?: string` — uncontrolled initial cycle.
- `cycleLabel?: string` — `aria-label` for the billing cycle `ToggleGroup` (default `'Billing period'`).
- `cols?: number | { narrow?: number; regular?: number; wide?: number }` — override grid columns (defaults to `{ narrow: 1, regular: Math.min(plans.length, 2), wide: Math.min(Math.max(plans.length, 1), 4) }`).
- `tone?: Tone` — accent tone for the highlighted plan and checkmarks (`'primary'` by default).
- `align?: 'start' | 'center'` — alignment of the section header (`'center'` by default).
- `level?: 2 | 3 | 4 | 5 | 6` — section heading level (default `2`; plan titles are one level deeper).

**Emits:**
- `'update:cycle': [cycle: string]` — when the user picks a billing cycle.

**Slots:**
- `headline?: () => unknown`
- `header?: () => unknown`
- `plan?: (props: { plan: PricingPlan; index: number; cycle: string | undefined }) => unknown`
- `[name: string]: ((props: { plan: PricingPlan; index: number; cycle: string | undefined }) => unknown) | undefined`
- `footer?: () => unknown` — a note or enterprise link below the plans.

**Per-cycle map missing-cycle rule (documented in `Pricing.docs.md` and tested):**
- A plain string (`price: '$0'`, `period: '/mo'`) applies to every cycle.
- If `price` (or `period`) is a `Record<string, string>` and lacks the selected `cycle`, `Pricing` shows no price (the plan remains, its price area empty) so a monthly price is never shown under `"Yearly"`, and warns once in development naming the plan and the cycle (`Pricing: plan "<name>" has no price for cycle "<cycle>"; give it an entry or use a plain string for every cycle.`), never in production.
- For `billingNote`, if a `Record<string, string>` has no entry for the active `cycle` (for example, `{ yearly: 'Billed $288 yearly' }` with no note on `'monthly'`), no billing note is shown for that cycle (`''`).

**Adaptation:**
- `input`: the `ToggleGroup` cycle buttons and plan CTA buttons (`block` full-width buttons) are at least `--weft-hit-min` (44 px) tall on touch.
- `surface`: follows the surface's type scale and container width.
- `width`: plans stack in 1 column when `narrow`, 2 when `regular`, and up to `plans.length` (capped at 4) when `wide`.

---

### 4. `Testimonials` (`packages/ui/src/landing/Testimonials.vue`, `styles/testimonials.css`)

**Purpose:** Quotes from customers or users, each in a semantic `figure` with a `blockquote` and `figcaption` naming the author, their role or company, and their `Avatar`.

**Types (`packages/ui/src/landing/types.ts`):**
```ts
export interface TestimonialItem {
  /** The words quoted. */
  quote: string
  /** Who said it ("Ada Lovelace"). */
  author: string
  /** Their title, company or context ("CTO at Analytical Engine"). */
  role?: string
  /** Optional avatar image address; falls back to initials from `author`. */
  avatar?: string
  /** Optional link on the author or source. */
  to?: string
}
```

**Props:**
- `headline?: string`
- `title?: string`
- `description?: string`
- `items?: TestimonialItem[]`
- `cols?: number | { narrow?: number; regular?: number; wide?: number }` — columns per width class (default `{ narrow: 1, regular: 2, wide: 3 }`).
- `align?: 'start' | 'center'` — alignment of the section header (`'start'` by default).
- `level?: 2 | 3 | 4 | 5 | 6` — section heading level (default `2`).

**Slots:**
- `headline?: () => unknown`
- `header?: () => unknown`
- `item?: (props: { item: TestimonialItem; index: number }) => unknown`
- `default?: () => unknown`

**Adaptation:**
- `input`: non-interactive unless an item has `to`, which renders with `Link`.
- `surface`: follows the surface's type scale.
- `width`: reflows from 1 column (`narrow`) to 2 (`regular`) to 3 (`wide`) via `Grid`.

---

### 5. `Faq` (`packages/ui/src/landing/Faq.vue`, `styles/faq.css`)

**Purpose:** A titled Frequently Asked Questions section pairing a section header (and optional contact note) with an accessible `Accordion` whose answers stay searchable by find-in-page (`hidden="until-found"`).

**Types (`packages/ui/src/landing/types.ts`):**
- `export type FaqItem = AccordionItem` — one item shape shared with `Accordion` (`{ label, content, value, icon, trailingIcon, disabled, slot }`), so an `AccordionItem[]` works in both and there is only one way to write it.

**Props:**
- `headline?: string`
- `title?: string` — section heading (`h2` by default, or `h${level}`).
- `description?: string` — lead under the title.
- `items?: FaqItem[]` — the questions (`label`) and answers (`content`).
- `type?: 'single' | 'multiple'` — forwarded to `Accordion` (`'single'` by default).
- `modelValue?: string | string[] | null` — open item(s), bound with `v-model`.
- `defaultValue?: string | string[] | null` — initially open item(s) when `v-model` is not bound.
- `collapsible?: boolean` — with `single`, allows closing the open question (`true` by default).
- `orientation?: 'vertical' | 'horizontal'` — `'vertical'` (default: heading above the questions) or `'horizontal'` (heading and description in a column beside the questions when `wide`, stacked above when `narrow` or `regular`).
- `align?: 'start' | 'center'` — alignment of the header when `orientation="vertical"` (`'start'` by default).
- `level?: 2 | 3 | 4 | 5 | 6` — section heading level (default `2`; question headings in the `Accordion` render at `Math.min(level + 1, 6)`, so `h3` under an `h2`).

**Emits:**
- `'update:modelValue': [value: string | string[] | null]`

**Slots:**
- `headline?: () => unknown`
- `header?: () => unknown`
- `aside?: () => unknown` — extra content under the header (such as `"Still have questions? Contact us"`).
- `label?: (props: { item: FaqItem; index: number; open: boolean }) => unknown` — forwarded to `Accordion`'s `label` slot.
- `content?: (props: { item: FaqItem; index: number; open: boolean }) => unknown` — forwarded to `Accordion`'s `content` slot.
- `[name: string]: ((props: { item: FaqItem; index: number; open: boolean }) => unknown) | undefined` — named slot matching `item.slot`.

**Adaptation:**
- `input`: each question header is a full-width button at least `--weft-hit-min` (44 px on touch) tall, with full ArrowUp/ArrowDown/Home/End roving focus from `Accordion`.
- `surface`: follows the surface's type scale.
- `width`: measured from its own root via `useWidth(root)`; a `horizontal` FAQ becomes a two-column grid (`5fr 7fr`) when `wide` and a single stacked column when `narrow` or `regular`.

---

### 6. `Cta` (`packages/ui/src/landing/Cta.vue`, `styles/cta.css`)

**Purpose:** A closing call-to-action band or card that invites the reader to take the next step (start a trial, run an install command, talk to sales), with optional media or an install snippet beside or below the copy.

**Props:**
- `headline?: string` — eyebrow text above the title.
- `title?: string` — the call-to-action heading (`h2` by default, or `h${level}`).
- `description?: string` — supporting sentence under the title.
- `actions?: ActionItem[]` — buttons; the first defaults to `variant="solid"`, the rest to `variant="outline"`. (When `Cta`'s `variant="solid"`, buttons default to `tone="neutral"` so they contrast with the solid band; otherwise `tone` on the first button.)
- `variant?: 'card' | 'soft' | 'solid' | 'plain'` — `'card'` (default: a bordered `--weft-color-surface-raised` panel with `--weft-radius-panel`), `'soft'` (tinted in `tone`), `'solid'` (filled with `--weft-tone` and `--weft-tone-fg` text), or `'plain'` (no box).
- `tone?: Tone` — tone for `'soft'` and `'solid'` variants and the primary action (`'primary'` by default).
- `orientation?: 'vertical' | 'horizontal'` — `'vertical'` (default: copy and actions stacked) or `'horizontal'` (copy at the inline start, actions or `aside` slot at the inline end when `regular` or `wide`, stacked when `narrow`).
- `align?: 'start' | 'center'` — text and action alignment when `orientation="vertical"` (`'center'` by default when `vertical`, `'start'` when `horizontal`).
- `level?: 2 | 3 | 4 | 5 | 6` — heading level (default `2`).

**Slots:**
- `headline?: () => unknown`
- `title?: () => unknown`
- `description?: () => unknown`
- `actions?: () => unknown` — in place of `actions` (e.g. a `CopyCommand` island or newsletter form).
- `aside?: () => unknown` — an illustration or picture beside the copy (like the loom image in weft.base.al's closing band).
- `default?: () => unknown` — extra body content under the description.

**Adaptation:**
- `input`: action buttons are at least `--weft-hit-min` (44 px on touch) tall.
- `surface`: follows the surface's type scale and container width.
- `width`: measured from its own root via `useWidth(root)`; a `horizontal` CTA stacks into one column when `narrow`.
