# Parity with Nuxt UI 4.13.0

Nuxt UI 4.13.0 is Weft's benchmark for component coverage and quality. This
file maps every component the `@nuxt/ui@4.13.0` npm package ships
(`dist/runtime/components`, and its `color-mode`, `content`, `locale` and
`prose` folders) to Weft's equivalent and its status:

- **done:** Weft covers what the Nuxt component does.
- **partial:** Weft has the component; the props or features named are missing. Where the gap is by design, the notes say why.
- **planned (phase N):** a later phase of the spec (`docs/superpowers/specs/2026-11-20-weft-design.md`, Phases) builds it.
- **not planned:** not on Weft's roadmap, with the reason.

Across every row, some Nuxt props are styling plumbing that Weft does differently, and are not listed as gaps:
`as`, `class` and `ui` (Weft styles with `--weft-*` tokens and one stylesheet per component);
`color` (Weft's shared `tone`: `neutral`, `primary`, `danger`, `warning`, `success`);
`portal` and `content` (overlays always go to `#weft-overlays`);
`highlight` and `fixed` (field highlight styling).
"Icon overrides" means the `*Icon` props that swap a default glyph (`closeIcon`, `prevIcon`, …).
The gaps come from comparing each component's own props in Nuxt's `<Name>.vue.d.ts` with `catalog/components.json`, and from the parity notes in the components plan's task reports.

Every Weft component adapts to touch, pointer, landing and app inside itself, which Nuxt leaves to the page: popovers and menus become bottom sheets on touch, tooltips step aside, dialogs go full screen when narrow, hit targets reach 44 px.

## Components

| Nuxt UI | Weft | Status | Notes |
| --- | --- | --- | --- |
| Accordion | `Accordion` | partial: no `valueKey`, `labelKey`, `unmountOnHide` | Content is always rendered, by design, so find-in-page reaches it (`hidden="until-found"`). |
| Alert | `Alert` | partial: no `avatar`, `closeIcon` | `color` is `tone`. Defaults to `variant="soft"` (Nuxt: solid). |
| App | `WeftProvider` | partial: no `tooltip`, `toaster`, `portal` | `locale` is on WeftProvider; the toaster is its own component (`Toaster`), and overlays always go to `#weft-overlays`. |
| AuthForm | — | planned (Warp W4) | `warp/auth` signs people in; no login-form component yet: `examples/shop` writes its own on `Form` with the schema `weft gen` writes. |
| Avatar | `Avatar` | partial: no `chip`, `color` | Sizes are the shared `sm`/`md`/`lg` (Nuxt: 3xs to 3xl). Put a `Chip` around it for the dot. |
| AvatarGroup | `AvatarGroup` | done | Nuxt's `color` has no Avatar counterpart in Weft. |
| Badge | `Badge` | partial: no `square` | `color` is `tone`. |
| Banner | `Banner` | partial: no `target`, `closeIcon` | A banner closed under an `id` stays closed across reloads, set before paint (`usePrePaint`). |
| BlogPost | — | planned (phase 6b) | Content collections. |
| BlogPosts | — | planned (phase 6b) | Content collections. |
| Breadcrumb | `Breadcrumb` | partial: no `labelKey`, avatar on items | Middle items do not collapse into "…" when narrow; the trail scrolls instead. |
| Button | `Button` | partial: no loadingAuto for a promise from `@click`, `activeColor`, `activeVariant`, `avatar` | A submit Button shows its Form's busy state, as `loadingAuto` does there; `active` marks it (and `aria-current` on a link); `color` is `tone`. |
| Calendar | `Calendar` | partial: no `pagedNavigation`, `maximumDays`, `allowNonContiguousRanges`, month-cell and year-cell slots, icon overrides | Gregorian grid only (names and digits follow the locale). `type` month and year pickers are there. A single date does not deselect on a second press. |
| Card | `Card` | partial: no `title`, `description`, `variant` | Header, actions and footer are slots. |
| Carousel | — | planned (phase 7) | Landing blocks. |
| ChangelogVersion | — | planned (phase 6b) | Content collections. |
| ChangelogVersions | — | planned (phase 6b) | Content collections. |
| Chart | — | planned (phase 5) | Charts on `@unovis/ts` (`@weft/ui/charts`): line, area, bar, donut and sparkline, with live data. |
| ChatAttachment | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatMessage | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatMessages | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatPalette | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatPrompt | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatPromptSubmit | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatReasoning | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatShimmer | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| ChatTool | — | not planned | The AI chat family belongs to a later block plan; it needs a streaming chat model on the Go side first. |
| Checkbox | `Checkbox` | partial: no variant (card), `indicator`, `trueValue`, `falseValue`, icon overrides | Indeterminate is supported; `color` is `tone`. |
| CheckboxGroup | `CheckboxGroup` | partial: no `valueKey`, `labelKey`, `descriptionKey`, variant (card, table), `loop`, `required` |  |
| Chip | `Chip` | partial: no `standalone` |  |
| Collapsible | `Collapsible` | partial: no `unmountOnHide` | Content is always rendered, by design, for find-in-page. |
| ColorPicker | — | not planned | Not in the spec's component tiers; `<Input type="color">` gives the platform picker. |
| CommandPalette | `CommandPalette` | partial: no `fuse`, `multiple`, input: false, `searchDelay`, per-group slot, icon overrides | A built-in fuzzy matcher and `limit` replace Fuse.js; recent items are built in. |
| Container | `Container` | done |  |
| ContextMenu | `ContextMenu` | partial: no `size`, `modal`, `labelKey`, `descriptionKey`, icon overrides | Long press opens it on touch, as a bottom sheet. |
| ContextMenuContent | `ContextMenu` | done | An internal part of ContextMenu, not a component of its own. |
| DashboardGroup | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardNavbar | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardPanel | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardResizeHandle | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardSearch | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardSearchButton | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardSidebar | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardSidebarCollapse | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardSidebarToggle | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| DashboardToolbar | — | not planned | The Dashboard family belongs to a later block plan. `AppShell` covers the shell it builds (sidebar on wide, bottom bar on narrow touch, collapse); panels, resize handles and the dashboard search are that plan's. |
| Drawer | `Drawer` | partial: no `snapPoints`, `activeSnapPoint`, `nested`, `inset`, `handleOnly`, `shouldScaleBackground`, `modal` | Swipe to dismiss and `handle` are there; `side` picks the edge. |
| DropdownMenu | `DropdownMenu` | partial: no `size`, `modal`, `arrow`, `filter`, `filterFields`, `labelKey`, `descriptionKey`, `disabled`, icon overrides | Checkbox and radio items, submenus, typeahead and shortcut text (kbds) are there; a bottom sheet on touch. |
| DropdownMenuContent | `DropdownMenu` | done | An internal part of DropdownMenu, not a component of its own. |
| Editor | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| EditorDragHandle | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| EditorEmojiMenu | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| EditorMentionMenu | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| EditorSuggestionMenu | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| EditorToolbar | — | not planned | The rich-text Editor family (TipTap) belongs to a later block plan; Weft adds no dependency that needs the virtual DOM. |
| Empty | `EmptyState` | partial: no `avatar` |  |
| Error | `ErrorState` | partial: no `redirect`, `clear` | Nuxt's error page helpers; ErrorState takes an `error` and offers `retry`. |
| FieldGroup | `FieldGroup` | done |  |
| FileUpload | `FileUpload` | partial: no variant (button), `fileIcon`, `fileImage`, `fileDelete`, files-top, files-bottom and actions slots, icon overrides | `layout` (list, grid), `position` (inside, outside) and `reset` are there; on touch, one button offers camera, photos or files. |
| Footer | `Footer` | done |  |
| FooterColumns | — | planned (phase 7) | Landing blocks (the spec's `Footer` block). |
| Form | `Form` | partial: no `validate`, `validateOn`, `validateOnInputDelay`, `transform`, `nested`, Standard Schema validators | Validates against Weft's field schema, the shape `weft gen` writes as `schema` from the Go types; `onSubmit` and `loadingAuto` as in Nuxt. |
| FormField | `FormField` | partial: no `help`, `errorPattern`, `eagerValidation`, `validateOnInputDelay`, `orientation` |  |
| Header | `Header` | partial: no `mode`, `toggleSide`, `autoClose`, `toggle`, title and toggle slots |  |
| Icon | `Icon` | done | Lucide, inlined at build: only the icons a build uses ship. Iconify's `mode` and `customize` do not apply. |
| Input | `Input` | partial: no `variant`, `defaultValue`, `autofocus`, `autofocusDelay`, `modelModifiers`, `avatar`, `highlight` |  |
| InputDate | `DatePicker` | partial: no segmented date field, `separatorIcon` | A different design, as the plan asked: a text field parsed in the locale with a popover calendar (a bottom sheet on touch). `DateRangePicker` is its `range`. |
| InputMenu | `Combobox` | partial: no `autofocus`, `autofocusDelay`, `arrow`, `modelModifiers`, content-top and content-bottom slots, icon overrides | `trigger="input"`, the default: search, create option and async items. |
| InputNumber | `InputNumber` | partial: no `orientation`, `formatOptions`, `stepSnapping`, `invertWheelChange`, increment and decrement customisation, press-and-hold repeat, `autofocus` |  |
| InputRating | — | not planned | Not in the spec's component tiers; a `RadioGroup` of stars covers it until one is asked for. |
| InputTags | `InputTags` | partial: no `variant`, `autofocus`, `autofocusDelay`, `deleteIcon` |  |
| InputTime | `InputTime` | partial: no `isTimeUnavailable`, `separatorIcon` | `range` is there; on touch it is the platform time input. |
| Kbd | `Kbd` | partial: no `variant`, `color` |  |
| Link | `Link` | partial: no `exact`, `exactQuery`, `exactHash`, `activeClass`, `inactiveClass`, `custom`, `raw` | Weft has no client router: `active` is given, not matched against a route. |
| LinkBase | `Link` | done | An internal part of Link. |
| Listbox | `Listbox` | partial: no `orientation`, `highlightOnHover`, `modelModifiers` |  |
| Main | `Page` | done | The main region of a page. |
| Marquee | — | planned (phase 7) | Landing blocks. |
| Modal | `Dialog` | partial: no `transition` | Full screen when narrow; `overlay`, `scrollable`, `fullscreen`, `dismissible` as in Nuxt. |
| NavigationMenu | `NavigationMenu` | partial: no `delayDuration`, `disableHoverTrigger`, variant (pill, link), `highlight`, `contentOrientation`, `viewport`, `arrow`, tooltip and popover per item, `valueKey`, `labelKey`, v-model of open groups, children past one level | The APG disclosure-navigation pattern; a sheet list on narrow touch. |
| OverlayProvider | — | planned (phase 5) | Programmatic overlays (`useOverlay`) come with `EntityDialog`. Weft creates its `#weft-overlays` root when first needed, so no provider is placed. |
| Page | `Page` | partial: no left and right aside slots |  |
| PageAnchors | — | planned (phase 6b) | Docs pages from content collections. |
| PageAside | — | planned (phase 6b) | Docs pages from content collections. |
| PageBody | — | planned (phase 6b) | Docs pages from content collections. |
| PageCTA | — | planned (phase 7) | Landing blocks (`Cta`). |
| PageCard | — | planned (phase 7) | Landing blocks. |
| PageColumns | — | planned (phase 7) | Landing blocks. |
| PageFeature | — | planned (phase 7) | Landing blocks (`FeatureGrid`). |
| PageGrid | `Grid` | done | Columns follow the grid's own width. |
| PageHeader | — | planned (phase 6b) | Docs pages from content collections. |
| PageHero | — | planned (phase 7) | Landing blocks (`Hero`). |
| PageLinks | — | planned (phase 6b) | Docs pages from content collections. |
| PageList | — | planned (phase 7) | Landing blocks. |
| PageLogos | — | planned (phase 7) | Landing blocks. |
| PageSection | `Section` | partial: no `headline`, `icon`, `links`, `features`, `orientation`, `reverse` | The landing variants are phase 7 blocks. |
| Pagination | `Pagination` | partial: no icon overrides | `color` and `activeColor` are `tone` and `activeTone`. |
| PinInput | `PinInput` | partial: no `separator`, `autofocus`, `autofocusDelay` |  |
| Popover | `Popover` | done | Nuxt's `reference` is `anchor`; `mode="hover"`, delays and `arrow` are there. A bottom sheet on touch. |
| PricingPlan | — | planned (phase 7) | Landing blocks (`Pricing`). |
| PricingPlans | — | planned (phase 7) | Landing blocks (`Pricing`). |
| PricingTable | — | planned (phase 7) | Landing blocks (`Pricing`). |
| Progress | `Progress` | partial: no `inverted`, `orientation`, `animation`, steps (an array as max) |  |
| ProgressGroup | — | planned (phase 5) | A bar of segments (storage used by kind, say) goes with the charts; `Progress` draws one value. |
| RadioGroup | `RadioGroup` | partial: no `valueKey`, `labelKey`, `descriptionKey`, variant (card, table), `indicator`, `loop` |  |
| ScrollArea | `ScrollArea` | partial: no `items`, `virtualize`, scroll event | Table, Listbox and Tree window their own rows. |
| Select | `Select` | partial: no `arrow`, `modelModifiers`, `autofocus`, `autofocusDelay` | A bottom sheet with a search field on touch (`searchInput`). |
| SelectMenu | `Combobox` | partial: no `arrow`, `modelModifiers`, icon overrides | `trigger="button"`, with the search at the top of the popup. |
| Separator | `Separator` | partial: no `avatar`, `size`, type (dashed, dotted), `position` |  |
| Sidebar | `AppShell` | partial: no variant (floating, inset), side: right, collapsible: offcanvas or none, `rail`, resize handle, `mode`, toggle shortcut | Sidebar plus header when wide, bottom navigation on narrow touch, icon collapse remembered before paint. |
| Skeleton | `Skeleton` | done | Respects reduced motion. |
| Slideover | `Sheet` | partial: no `inset`, `transition` | `side` is any edge or the bottom sheet. |
| Slider | `Slider` | partial: no `tooltip`, `inverted` |  |
| Splitter | — | not planned | Resizable panels go with the Dashboard family's resize handle, in a later block plan. |
| Stepper | `Stepper` | partial: no `valueKey` |  |
| Switch | `Switch` | partial: no `checkedIcon`, `uncheckedIcon`, `trueValue`, `falseValue` |  |
| Table | `Table` | partial: no `filtering`, expandable rows, column visibility and pinning, `grouping`, sticky footer, row click and hover events, `meta`, virtualize options | Columns, sorting, selection, sticky header, pagination, `fetch`, a keyboard grid and stacked cards when narrow. TanStack-level features are the spec's `DataTable` (phase 5). Nuxt's `data` is `rows`. |
| Tabs | `Tabs` | partial: no `valueKey`, `labelKey`, avatar on items | `unmountOnHide` is `keepAlive` inverted; `hash` syncs the URL. |
| Textarea | `Textarea` | partial: no `variant`, `defaultValue`, `autoresizeDelay`, `autofocus`, `modelModifiers` |  |
| Theme | — | not planned | A Weft theme is a token file (`@weft/themes`); a part of a page is re-themed with `--weft-*` custom properties on any ancestor, so no component is needed. |
| Timeline | `Timeline` | partial: no `valueKey`, select event, per-item slots |  |
| Toast | `Toast` | partial: no `avatar`, `orientation` | `progress`, actions, swipe and pause on hover or focus are there. |
| Toaster | `Toaster` | partial: no `expand`, `progress` | Progress is per toast. F8 reaches it, even while a modal is open. |
| Toggle | `Toggle` | done | `activeColor` is `activeTone`. |
| ToggleGroup | `ToggleGroup` | partial: no `type`, `rovingFocus`, `valueKey`, `labelKey` | `multiple` in place of `type`; roving focus is always on. |
| Tooltip | `Tooltip` | partial: no `kbds`, `arrow`, `reference` | Renders nothing on touch; its text names an unnamed trigger. |
| Tree | `Tree` | partial: no `getKey`, `selectionBehavior`, `propagateSelect`, `bubbleSelect`, `nested`, virtualize options, `loop`, icon overrides | Its `v-model` holds keys, not items; it windows itself past 200 rows. |
| User | — | not planned | Not in the spec's tiers: an `Avatar` beside a title and description in a `Stack` covers it. |

## `color-mode/`

| Nuxt UI | Weft | Status | Notes |
| --- | --- | --- | --- |
| ColorModeAvatar | — | not planned | Not a component in Weft: the scheme is `data-weft-scheme` on the root, set before paint (`usePrePaint`); a `Switch` or `Select` drives it. |
| ColorModeButton | — | not planned | Not a component in Weft: the scheme is `data-weft-scheme` on the root, set before paint (`usePrePaint`); a `Switch` or `Select` drives it. |
| ColorModeImage | — | not planned | Not a component in Weft: the scheme is `data-weft-scheme` on the root, set before paint (`usePrePaint`); a `Switch` or `Select` drives it. |
| ColorModeSelect | — | not planned | Not a component in Weft: the scheme is `data-weft-scheme` on the root, set before paint (`usePrePaint`); a `Switch` or `Select` drives it. |
| ColorModeSwitch | — | not planned | Not a component in Weft: the scheme is `data-weft-scheme` on the root, set before paint (`usePrePaint`); a `Switch` or `Select` drives it. |

## `content/`

| Nuxt UI | Weft | Status | Notes |
| --- | --- | --- | --- |
| ContentNavigation | — | planned (phase 6b) | Content collections (docs navigation, search, table of contents). |
| ContentSearch | — | planned (phase 6b) | Content collections (docs navigation, search, table of contents). |
| ContentSearchButton | — | planned (phase 6b) | Content collections (docs navigation, search, table of contents). |
| ContentSurround | — | planned (phase 6b) | Content collections (docs navigation, search, table of contents). |
| ContentToc | — | planned (phase 6b) | Content collections (docs navigation, search, table of contents). |

## `locale/`

| Nuxt UI | Weft | Status | Notes |
| --- | --- | --- | --- |
| LocaleSelect | — | not planned | Locale switching is app work; WeftProvider carries the `locale` components format with, and a `Select` lists the app's locales. |

## `prose/`

| Nuxt UI | Weft | Status | Notes |
| --- | --- | --- | --- |
| ProseA | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseAccordion | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseAccordionItem | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseBadge | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseBlockquote | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCallout | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCard | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCardGroup | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCaution | — | planned (phase 6b) | Content collections render Markdown with prose components; an `Alert` with a tone draws the same callout today. |
| ProseCode | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCodeCollapse | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCodeGroup | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCodeIcon | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCodePreview | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCodeTree | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseCollapsible | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseEm | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseField | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseFieldGroup | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseH1 | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseH2 | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseH3 | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseH4 | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseHr | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseIcon | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseImg | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseKbd | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseLi | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseNote | — | planned (phase 6b) | Content collections render Markdown with prose components; an `Alert` with a tone draws the same callout today. |
| ProseOl | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseP | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProsePre | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProsePrompt | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseScript | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseSteps | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseStrong | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTable | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTabs | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTabsItem | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTbody | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTd | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTh | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseThead | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseTip | — | planned (phase 6b) | Content collections render Markdown with prose components; an `Alert` with a tone draws the same callout today. |
| ProseTr | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseUl | — | planned (phase 6b) | Content collections render Markdown with prose components. |
| ProseWarning | — | planned (phase 6b) | Content collections render Markdown with prose components; an `Alert` with a tone draws the same callout today. |

## Weft components without a Nuxt UI counterpart

`Stack`, `Divider`, `Island`, `Spinner`, `ConfirmDialog`, `DateRangePicker`, `StatCard`, `CloseButton`, `OverlayPanel`. (`DateRangePicker` is what `InputDate`'s `range` does in Nuxt; `Divider` is the layout rule beside `Separator`; `CloseButton` and `OverlayPanel` are the parts every Weft overlay is built from, public so pages can build their own.)

## Summary

186 Nuxt UI components: 128 at the top level, 58 in its folders (`prose/` holds its four callouts, Caution, Note, Tip and Warning, in `prose/callout/`).

| Status | Count |
| --- | ---: |
| done | 13 |
| partial | 59 |
| planned | 78 |
| not planned | 36 |
