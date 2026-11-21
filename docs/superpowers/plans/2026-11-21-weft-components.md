# Weft Components (overnight) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This plan is built to run unattended overnight: no task publishes, pushes to a live server or deletes owner files.

**Goal:** The actual Weft component library. That means Weft's own headless layer, plus the components Nuxt UI 4.13.0 ships for general apps: basics, forms, overlays, selection, dates and files, navigation, feedback and data. Each component adapts to touch, pointer, landing and app, and each carries the docs and examples the `/components` gallery will read.

**Benchmark (owner, 2026-11-21):** "nuxt and nuxtui is benchmark". Nuxt UI 4.13.0 is the bar for component coverage and quality; Nuxt is the bar for app features in later plans. Task 9 writes `docs/parity.md`, mapping every Nuxt UI 4.13.0 component to Weft's status.

**Architecture:**
- **Headless layer:** `packages/ui/src/headless/` holds composables for focus, dismiss, roving focus, positioning, typeahead, ids, controlled models, the overlay stack and scroll lock. No Reka, and no other UI library.
- **Components:** each one is a `<script setup vapor>` SFC in `packages/ui/src/<group>/`. It reads only `--weft-*` tokens, has one stylesheet in `src/styles/<group>.css` bundled into `weft.css`, a `<Name>.docs.md` beside it, and at least one example in `packages/ui/examples/<Name>/*.vue`.
- **Catalogue:** `scripts/catalog.ts` generates `catalog/components.json` from the SFCs and docs files. A test keeps it fresh.

**Tech Stack:**
- Vue `3.6.0-rc.16` (Vapor) and Vitest with happy-dom, following the existing `packages/ui` patterns.
- `@floating-ui/dom 1.8.0` for positioning and `@internationalized/date 3.12.4` for calendars.
- `lucide-static 1.77.0` for icons, inlined at build: only the icons a build uses ship.
- `@vue/compiler-sfc` matching the pinned Vue, used by the catalogue script.
- Re-check every version with `npm view` before adding it.

**Spec:** `docs/superpowers/specs/2026-11-20-weft-design.md`: Components, The adaptive model, Vapor, phase 3.

## Global Constraints

**Vue and Vapor**
- Every component is `<script setup vapor lang="ts">`. No Options API, `getCurrentInstance`, `globalProperties`, `v-memo`, VNode render functions or `vaporInteropPlugin`.

**Component API**
- Names are plain PascalCase: `Button`, `DatePicker`.
- The main value is bound with `v-model`.
- Prop names are shared across components:
  - `size`: `sm` / `md` / `lg`
  - `tone`: `neutral` / `primary` / `danger` / `warning` / `success`
  - `variant`: only where a component has real visual variants (`solid` / `soft` / `outline` / `ghost` for buttons and badges)
  - also `loading`, `disabled`, `icon` (a Lucide name string)
- Slot names are shared too: `header`, `footer`, `actions`, `empty`, `leading`, `trailing`.

**Adaptive rules**
- Adaptive behaviour lives inside the component and reads `useWeft()` and `useWidth()`. There is never a `Mobile*` variant.
- Touch rules:
  - every hit target is at least `--weft-hit-min` (44 px)
  - popovers and menus become bottom sheets
  - tooltips are not shown
  - dialogs become full-screen when narrow
- These touch rules are tested per component.

**Styling**
- Styles use only `--weft-*` tokens, each with a visible fallback.
- New tokens go in `@weft/themes` (in `themeToCss` and both themes), with tests.
- Each new group stylesheet is imported from `src/styles/weft.css`.

**Accessibility**
- Follow the WAI-ARIA Authoring Practices pattern for each widget.
- Every interactive component is tested for keyboard behaviour, roles and aria state, and visible focus.

**Docs and catalogue**
- Every component has `<Name>.docs.md`, with frontmatter `group`, `purpose`, `use`, `avoid` and `adapts` (per axis, in prose), plus at least one example.
- `pnpm --filter @weft/ui test` includes the catalogue-freshness test.

**Tests and process**
- Test first: the failing test is seen before the code. Mutation-check the key tests.
- Never delete owner files; tests may delete their own temp dirs.
- No `Co-Authored-By` in commits. Commits end with `Claude-Session: https://claude.ai/code/session_011JKtgodRX7P34o3jKTnB8d`.

**Size**
- Every component's CSS plus JS must stay small. Task 9 reports a per-component brotli size table, and there is no hard cap per component.
- Importing one component must not pull in the others: check tree-shaking in Task 9.

## Review Focus

1. **Overlays stacked three deep** (a dialog opening a menu opening a tooltip): Escape closes only the top one, focus returns to the element that opened each layer, and scroll lock is released only when the last overlay closes.
2. **Live switch from pointer to touch while an overlay is open:** it re-renders in the new form without losing state or focus.
3. **Disabled and loading states:** they block activation by mouse, keyboard and form submit alike, and `aria-disabled` or `aria-busy` is set.
4. **Right-to-left documents** (`dir="rtl"` on an ancestor): arrow keys and placement mirror correctly in roving-focus widgets and floating elements.
5. **Server rendering:** every component renders on the server without touching `window` or `document`, and hydrates without a mismatch when used inside an `<Island>`.

---

### Task 1: Headless layer, Icon and the catalogue convention

**Files:**
- `packages/ui/src/headless/{useId,useControllable,useFocusTrap,useDismiss,useRovingFocus,useFloating,useTypeahead,useScrollLock,overlayStack}.ts`
- `packages/ui/src/basics/Icon.vue`
- `packages/vite/src/icons.ts`, a Vite plugin that inlines the used Lucide SVGs as a virtual module
- `scripts/catalog.ts`, `catalog/components.json`
- `packages/ui/test/headless/*.test.ts`
- the catalogue test

**Interfaces:**
- `useId(prefix?: string): string`: SSR-stable. Use Vue's `useId` if it works in Vapor; otherwise a counter scoped to the app.
- `useControllable<T>(props, key, emit, default): Ref<T>`: v-model with an uncontrolled fallback.
- `useFocusTrap(el: Ref<HTMLElement|null>, opts: { active: Ref<boolean>; initial?: () => HTMLElement|null; returnTo?: HTMLElement|null })`.
- `useDismiss(el, { onDismiss, escape?: boolean, outside?: boolean })`: an Escape press goes to the top of `overlayStack` only.
- `useRovingFocus(container, { orientation: 'horizontal'|'vertical'|'grid', loop?: boolean, rtl?: Ref<boolean> })`.
- `useFloating(reference, floating, { placement, offset, flip, shift, size?, strategy? })`: wraps `@floating-ui/dom` `computePosition` and `autoUpdate`.
- `useTypeahead(items: () => string[], onMatch: (index: number) => void)`.
- `useScrollLock(active: Ref<boolean>)`: reference-counted.
- `overlayStack`:
  - `push(id)` and `pop(id)`, with `top()` and `isTop(id)` reactive;
  - `restoreFocus` behaviour.
- `<Icon name="trash" :size? :label?>`:
  - decorative (`aria-hidden`) unless `label` is given;
  - an unknown name fails the build with one sentence naming the file.
- `<Name>.docs.md` frontmatter schema and the example folder convention.
- `scripts/catalog.ts` uses `vue/compiler-sfc` `compileScript` to read props, defaults, emits and slots, and writes `catalog/components.json`. Entries are sorted, and the JSON shape is documented in the script header.

**Tests:**
- each composable: behaviour, SSR safety, cleanup on scope dispose;
- the overlay stack's Escape ordering;
- roving focus in RTL;
- Icon inlining in a real Vite build;
- a catalogue test that fails when a docs file or example is missing, or when the JSON is stale.

**Retrofit:** add docs and examples for the existing `Stack`, `Grid`, `Page`, `Section`, `Card`, `Divider`, `WeftProvider` and `Island`.

### Task 2: Basics
`Button`, `Link`, `Badge`, `Avatar`, `AvatarGroup`, `Kbd`, `Chip`, `Separator`, `Skeleton`, `Progress`, `Spinner`, `Container`.

- **`Button`:** `variant`, `tone`, `size`, `icon`, `trailing-icon`, `loading` (keeps its width, sets `aria-busy`, blocks clicks), `disabled`, `block`, `to` (renders an `<a>`), and `type`.
  - Touch: a minimum 44 px hit area, without changing the visual size, by using a pseudo-element.
- **`Link`:** an external link gets `rel="noopener"`, and the current page gets `aria-current`.
- **`Avatar`:** falls back from the image to initials to an icon.
- **`Progress`:** determinate and indeterminate, with `role="progressbar"`.
- **`Skeleton`:** respects reduced motion.

Tests cover each prop, and keyboard activation for `Button` and `Link`.

### Task 3: Form inputs
`Input`, `Textarea` (autoresize), `InputNumber` (step, min, max, arrow keys, wheel off by default), `Checkbox` (indeterminate), `CheckboxGroup`, `RadioGroup` (roving focus), `Switch`, `Slider` (keyboard: arrows, Page Up/Down, Home/End; ranges with two thumbs), `PinInput`, `FieldGroup`, `FormField`, `Form`.

- **`FormField`:** the label, hint, error and required marker, wired to its control with `aria-describedby`.
- **`Form`:** validates against a Weft field-schema object, the same shape the Go contract will generate later: `{ field: { required?, min?, max?, pattern?, email?, message? } }`. It shows errors on submit and on blur after the first submit, accepts server errors through `setErrors({ field: message })`, and emits `submit` with typed values.
- **Touch:** control height is `--weft-control-height` (44 px); `InputNumber` steppers become large buttons; `Slider` thumbs grow to 44 px.

Tests cover v-model, validation, aria wiring, the keyboard, and the touch rules.

### Task 4: Overlays
`Popover`, `Tooltip`, `Dialog` (modal), `Sheet` (slideover from a side, or a bottom sheet), `Drawer`, `ConfirmDialog`, `Toast` and `Toaster` (a `useToast()` queue with timeout `--weft-duration-toast`, pause on hover or focus, and an `aria-live` region), `DropdownMenu` (items, groups, checkbox and radio items, submenus, typeahead), `ContextMenu`.

- Everything uses the Task 1 layer: focus trap, dismiss, overlay stack and scroll lock.
- Content goes through `Teleport` to a `#weft-overlays` root, created when first needed.
- **Adaptive:**
  - On touch, `Popover`, `DropdownMenu` and `ContextMenu` render as a bottom `Sheet` with large items, and `Tooltip` renders nothing; its label becomes `aria-label` on the trigger if the trigger has none.
  - When narrow, `Dialog` goes full-screen.

Tests cover Review Focus 1 and 2 explicitly, and the menu keyboard pattern.

### Task 5: Selection
`Select` (a native-feeling listbox), `Combobox` (Nuxt's SelectMenu and InputMenu: search, create option, async items), `InputTags`, `Listbox`, `CommandPalette` (groups, fuzzy match, recent items, keyboard).

- **Adaptive:** on touch, `Select` and `Combobox` open as a bottom sheet with a search field at the top.
- Large lists are virtualized above 200 items, using a small virtualizer in `headless/`.

Tests cover the ARIA listbox and combobox patterns, typeahead, async loading states, and the empty slot.

### Task 6: Dates and files
`Calendar` (single, range, multiple; min, max, disabled dates; locale and first weekday from `@internationalized/date`), `DatePicker`, `DateRangePicker`, `InputTime`, `FileUpload`.

- **`DatePicker` and `DateRangePicker` adapt:** on pointer, an input with a popover calendar, where typed dates are parsed in the locale; on touch, a bottom sheet with large day targets and month swipe.
- **`FileUpload` adapts:** on pointer, a drop zone plus a browse button, with previews, progress per file and a remove action; on touch, one button offering camera, photos or files (through `accept` and `capture`), with the same list below.

Tests cover the calendar grid keyboard (arrows, Page Up/Down for months, Home/End), locale formatting, drop and paste for files, and the touch forms.

### Task 7: Navigation and disclosure
`Tabs` (manual or automatic activation, roving focus), `Breadcrumb`, `Pagination`, `NavigationMenu` (horizontal with dropdowns on pointer; a sheet list on narrow touch), `Stepper`, `Accordion`, `Collapsible`, `Tree` (with keyboard), `AppShell` (sidebar plus header when wide; a bottom navigation bar on narrow touch; sidebar collapse), `Header`, `Footer`.

Tests cover the ARIA patterns, current page and `aria-current`, the keyboard, and the adaptive layouts.

### Task 8: Feedback and data
`Alert`, `Banner`, `EmptyState`, `ErrorState`, `StatCard`, `Timeline`, `Toggle`, `ToggleGroup`, `ScrollArea`, `Table`.

- **`Table`:** columns, sorting, row selection, sticky header, an empty slot, a loading state, and client-side pagination by composing `Pagination`.
- **Adaptive:** when narrow, `Table` renders rows as stacked cards from the same column definitions.
- `StatCard` and `EmptyState` replace the hand-built Negenet equivalents named in the spec.

### Task 9: Parity, sizes and the gallery hand-off
1. **`docs/parity.md`:** every Nuxt UI 4.13.0 component (list it from the npm package in a temp dir) mapped to its Weft equivalent, "planned (phase N)", or "not planned" with a reason (e.g. the AI chat family, Editor, Dashboard, which belong to later block plans).
2. **`docs/sizes.md`:** brotli JS and CSS per component, from a Vite build of each example alone, plus a tree-shaking test showing that importing `Button` alone does not pull in `Dialog`.
3. **Catalogue:** `catalog/components.json` must be complete and fresh.
4. **Playground:** update it so every component's first example renders under the adaptive controls. Check it in headless Chrome at 1280 and 390 px, pointer and touch, light and dark, and save screenshots to `docs/assets/components-*.webp` (WebP, cropped to the first 2400 px).
5. **Review Focus 5:** an SSR-render test for every example, through `renderPage`, plus a hydrate-in-island test for the interactive ones.
