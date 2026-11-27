# Landing Blocks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `@weft/ui`'s six landing blocks (`Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta`) with their stylesheets, docs, examples, catalogue entries, parity and size tables, and unit + SSR/hydration tests.

**Architecture:** Six Vapor components under `packages/ui/src/landing/`, each importing `../styles/base.css` (and shared/child component sheets) followed by its own `../styles/<name>.css`, grouped in `packages/ui/src/styles/landing.css` and `weft.css`. Each block measures its own container width with `useWidth(root)` for responsive reflow, uses `ActionItem` and `Button` for actions, uses `orientation: 'vertical' | 'horizontal'` for stacked vs side-by-side layout, and respects `useInset()` / `level` for heading hierarchy.

**Tech Stack:** Vue `3.6.0-rc.16` (`<script setup vapor lang="ts">`), Vitest with happy-dom, plain CSS on `--weft-*` tokens.

**Spec:** `docs/superpowers/specs/2026-11-27-landing-blocks-design.md`.

## Global Constraints

- **Inherited:** every rule in `AGENTS.md` and the Global Constraints of `docs/superpowers/plans/2026-11-21-weft-components.md`.
- **Vapor only:** every `.vue` file starts `<script setup vapor lang="ts">`. Never use the Options API, `getCurrentInstance`, `app.config.globalProperties`, `v-memo`, VNode render functions or `vaporInteropPlugin`.
- **Tokens only:** styles use only `--weft-*` custom properties, each with a fallback.
- **Adaptive from inside:** width comes from `useWidth(root)` on the block's own element, never `window.innerWidth`. Every interactive target reaches `--weft-hit-min` (44 px on touch).
- **Stylesheet cascade order:** each component imports `../styles/base.css` first, any child component before its own sheet, and its own `../styles/<name>.css` last; `landing.css` lists the sheets in the exact order components import them (`packages/ui/test/stylesheets.test.ts` checks this).
- **Never edit generated files by hand:** run `pnpm catalog` for `catalog/components.json` and `node scripts/sizes.ts` for `docs/sizes.md`.
- **Never delete with `rm`:** use `trash` if needed, and never delete owner files.
- **No `Co-Authored-By`** or attribution lines in commits.
- **Work in** `.worktrees/landing-blocks`, branch `landing-blocks`.

## Review Focus

1. **Heading outline & landmarks:** `Hero` renders an `h1` by default, an `h2` inside `<WeftProvider inset>` (so the gallery stage and page previews never add a second `h1` to the host page), and `h${level}` when `level` is passed. `FeatureGrid`, `Pricing`, `Testimonials`, `Faq` and `Cta` render `h2` by default (or `h${level}`), so no block renders a second `h1` when several blocks are on one page, and child item headings (`FeatureGrid` items, `Pricing` plans, `Faq` accordion triggers) render one level deeper (`Math.min(level + 1, 6)`).
2. **Container-width adaptation (`useWidth`) & `orientation`:** `Hero`, `Faq` and `Cta` all use `orientation: 'vertical' | 'horizontal'` with `'vertical'` as the fixed default (and `align: 'start' | 'center'` for text alignment). Resizing a block's container between `narrow` (< 640 px), `regular` (< 1024 px) and `wide` (≥ 1024 px) updates `data-width` and reflows `Hero` (`horizontal` → stacked when not `wide`), `FeatureGrid` (`Grid` cols), `Pricing` (plan columns), `Testimonials` (`Grid` cols), `Faq` (`horizontal` → stacked when not `wide`) and `Cta` (`horizontal` → stacked when `narrow`), even when the viewport is wide.
3. **Interactive accessibility & touch targets:**
   - `Pricing` with `cycles` renders a `ToggleGroup` (`role="group"`, `aria-pressed` buttons, roving focus with Left/Right/Up/Down/Home/End) that switches prices, periods and billing notes live, emits `update:cycle`, and handles a cycle missing from a per-cycle map (`price` and `period` fall back to the map's first value; `billingNote` shows nothing).
   - `Faq` uses `AccordionItem` (`FaqItem = AccordionItem`: `{ label, content, value, icon, trailingIcon, disabled, slot }`) and delegates to `Accordion` with `hidden="until-found"` panels, `aria-expanded`, `aria-controls`, `role="region"` and roving focus across question buttons.
   - All action buttons, cycle switches and FAQ triggers meet `--weft-hit-min` (44 px) on touch.
4. **Server rendering & tree-shaking:** every example renders on the server without touching `window` or `document`, hydrates cleanly when interactive, and `packages/ui/test/stylesheets.test.ts` + `packages/vite/test/tree-shaking.test.ts` pass.

---

### Task 1: Shared landing types, `Hero` and `Cta`

**Files:**
- Create `packages/ui/src/landing/types.ts`
- Create `packages/ui/src/landing/Hero.vue`, `packages/ui/src/landing/Hero.docs.md`, `packages/ui/src/styles/hero.css`
- Create `packages/ui/src/landing/Cta.vue`, `packages/ui/src/landing/Cta.docs.md`, `packages/ui/src/styles/cta.css`
- Create `packages/ui/examples/Hero/basic.vue`, `packages/ui/examples/Hero/horizontal.vue`
- Create `packages/ui/examples/Cta/basic.vue`, `packages/ui/examples/Cta/horizontal.vue`
- Create `packages/ui/test/landing.test.ts` and `packages/ui/test/fixtures/LandingCases.vue`

- [ ] **Step 1: Write failing tests** in `packages/ui/test/landing.test.ts` for `Hero` (default `h1`, `h2` inside `WeftProvider inset`, explicit `level`, `headline`, `description`, `actions` rendering `Button`s with solid first and outline rest, fixed default `orientation="vertical"` even with a default slot, explicit `orientation="horizontal"`, `align`, `useWidth` setting `data-width="narrow"` / `"wide"`, slots replacing props) and `Cta` (default `h2`, `level`, `variant` `card`/`soft`/`solid`/`plain`, `tone`, `actions`, fixed default `orientation="vertical"`, explicit `orientation="horizontal"`, `align`, `aside` slot, `data-width` from `useWidth`).
- [ ] **Step 2:** Run `pnpm --filter @weft/ui test test/landing.test.ts` and see it fail.
- [ ] **Step 3: Implement** `types.ts`, `Hero.vue`, `Hero.docs.md`, `hero.css`, `Cta.vue`, `Cta.docs.md`, `cta.css` and their examples.
- [ ] **Step 4:** Re-run `pnpm --filter @weft/ui test test/landing.test.ts` and see it pass.

### Task 2: `FeatureGrid` and `Testimonials`

**Files:**
- Create `packages/ui/src/landing/FeatureGrid.vue`, `packages/ui/src/landing/FeatureGrid.docs.md`, `packages/ui/src/styles/feature-grid.css`
- Create `packages/ui/src/landing/Testimonials.vue`, `packages/ui/src/landing/Testimonials.docs.md`, `packages/ui/src/styles/testimonials.css`
- Create `packages/ui/examples/FeatureGrid/basic.vue`, `packages/ui/examples/FeatureGrid/plain.vue`
- Create `packages/ui/examples/Testimonials/basic.vue`
- Modify `packages/ui/test/landing.test.ts` and `packages/ui/test/fixtures/LandingCases.vue`

- [ ] **Step 1: Write failing tests** in `packages/ui/test/landing.test.ts` for `FeatureGrid` (section `h2` and item `h3`, custom `level`, icons in discs with `tone`, `variant="card"` vs `"plain"`, `to` and `linkLabel` rendering `Link`, named per-item `slot`, responsive `cols` through `Grid` on `ResizeObserver`) and `Testimonials` (`figure` + `blockquote` + `figcaption`, `Avatar` with `src` or initials from `author`, `role`, optional `to` link, responsive `cols` through `Grid`).
- [ ] **Step 2:** Run `pnpm --filter @weft/ui test test/landing.test.ts` and see it fail.
- [ ] **Step 3: Implement** `FeatureGrid` and `Testimonials`, their stylesheets, docs and examples.
- [ ] **Step 4:** Re-run `pnpm --filter @weft/ui test test/landing.test.ts` and see it pass.

### Task 3: `Pricing` and `Faq`

**Files:**
- Create `packages/ui/src/landing/Pricing.vue`, `packages/ui/src/landing/Pricing.docs.md`, `packages/ui/src/styles/pricing.css`
- Create `packages/ui/src/landing/Faq.vue`, `packages/ui/src/landing/Faq.docs.md`, `packages/ui/src/styles/faq.css`
- Create `packages/ui/examples/Pricing/basic.vue`, `packages/ui/examples/Pricing/cycles.vue`
- Create `packages/ui/examples/Faq/basic.vue`, `packages/ui/examples/Faq/horizontal.vue`
- Modify `packages/ui/test/landing.test.ts` and `packages/ui/test/fixtures/LandingCases.vue`

- [ ] **Step 1: Write failing tests** in `packages/ui/test/landing.test.ts` for `Pricing` (plans rendered with title at `level + 1`, price, period, billingNote, features list with check icon, `highlight` and `badge`, CTA button solid on highlighted plan and outline on others; billing `cycles` rendered with `ToggleGroup` and cycle `badge`, keyboard arrow roving focus, switching cycle updates per-cycle price/period/billingNote and emits `update:cycle`, a plain string price applies to every cycle while a per-cycle map lacking the selected cycle shows an empty price area and warns once in development naming the plan and cycle, responsive columns via `useWidth`) and `Faq` (section heading at `level` and accordion headers at `level + 1`, `AccordionItem` `{ label, content }` items, `v-model` and `defaultValue`, `type="single"` and `"multiple"`, `hidden="until-found"` panels, keyboard toggling and roving focus, `orientation="vertical"` vs `"horizontal"` with `data-width`, `aside` slot).
- [ ] **Step 2:** Run `pnpm --filter @weft/ui test test/landing.test.ts` and see it fail.
- [ ] **Step 3: Implement** `Pricing` and `Faq`, their stylesheets, docs and examples.
- [ ] **Step 4:** Re-run `pnpm --filter @weft/ui test test/landing.test.ts` and see it pass.

### Task 4: Bundle wiring, catalogue, gallery, parity, sizes and full suite

**Files:**
- Create `packages/ui/src/styles/landing.css`
- Modify `packages/ui/src/styles/weft.css`
- Modify `packages/ui/src/index.ts`
- Modify `packages/ui/test/stylesheets.test.ts`
- Modify `site-src/gallery/model.ts`
- Modify `skills/weft-components/SKILL.md`, `site-src/content/docs/index.md`, `site-src/content/docs/coming-next.md`
- Modify `docs/parity.md`
- Regenerate `catalog/components.json` (`pnpm catalog`) and `docs/sizes.md` (`node scripts/sizes.ts`)

- [ ] **Step 1:** Wire `packages/ui/src/styles/landing.css`, `packages/ui/src/styles/weft.css`, `packages/ui/src/index.ts`, `packages/ui/test/stylesheets.test.ts`, and `site-src/gallery/model.ts`.
- [ ] **Step 2:** Update component counts (`76` → `82`) and group descriptions in `skills/weft-components/SKILL.md`, `site-src/content/docs/index.md`, `site-src/content/docs/coming-next.md`, and `docs/parity.md`.
- [ ] **Step 3:** Run `pnpm catalog` and `node scripts/sizes.ts`.
- [ ] **Step 4:** Run the full verification suite (`pnpm test`, `pnpm typecheck`, `scripts/playground.sh build`, `scripts/site.sh build`, `pnpm --filter weft-bench build`, `go vet ./...`, `go test -short ./...`, `(cd dbtest && go test ./...)`) and fix any issues.
