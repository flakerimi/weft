# Component sizes

What each `@weft/ui` component costs a page, measured by `node scripts/sizes.ts`
(which wrote this file). Each row is a Vite build of the component's first
example, `packages/ui/examples/<Name>/basic.vue`, alone: mounted with
`createVaporApp`, Vapor, production mode, minified, with `weftIcons()`, so
only the icons the example draws ship. Sizes are brotli at quality 11, as the
page build sends them, in KB (1024 bytes).

An example may draw more than its component: a Dialog example has a Button
to open it, a Form example its fields. The numbers are what that example
costs, which is what a page using the component in that way pays.

JS is what loads with the page; Later, what a component loads on demand
(a dynamic `import()`), such as the DatePicker's popover and calendar, fetched
when the pointer or focus first comes to it. Total is JS and CSS: what the
page pays before anyone touches it.

## Shared by every page

| What | Brotli |
| --- | ---: |
| Vue runtime (Vapor), its own chunk in each build | 17.40 to 26.23 KB |
| `styles/base.css`, inlined on every page by the page build | 0.31 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

76 components. Median: JS 2.55 KB, CSS 1.29 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.74 | 3.53 |
| Alert | 2.70 | — | 1.71 | 4.41 |
| AppShell | 18.48 | — | 3.68 | 22.15 |
| Avatar | 1.33 | — | 0.53 | 1.86 |
| AvatarGroup | 1.34 | — | 0.41 | 1.75 |
| Badge | 0.87 | — | 0.79 | 1.66 |
| Banner | 2.88 | — | 1.56 | 4.44 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.38 | — | 1.38 | 2.77 |
| Calendar | 10.99 | — | 1.31 | 12.30 |
| Card | 1.45 | — | 1.35 | 2.80 |
| Checkbox | 1.79 | — | 0.89 | 2.68 |
| CheckboxGroup | 2.35 | — | 0.99 | 3.33 |
| Chip | 1.55 | — | 1.08 | 2.63 |
| CloseButton | 0.62 | — | 0.52 | 1.14 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 21.99 | — | 2.56 | 24.55 |
| CommandPalette | 11.55 | — | 1.86 | 13.41 |
| ConfirmDialog | 6.53 | — | 1.88 | 8.40 |
| Container | 0.76 | — | 0.14 | 0.90 |
| ContextMenu | 14.51 | — | 1.77 | 16.28 |
| DatePicker | 12.13 | 14.78 | 2.61 | 14.74 |
| DateRangePicker | 25.35 | — | 2.62 | 27.97 |
| Dialog | 7.68 | — | 2.18 | 9.86 |
| Divider | 0.47 | — | 0.33 | 0.80 |
| Drawer | 6.77 | — | 2.21 | 8.98 |
| DropdownMenu | 14.83 | — | 2.21 | 17.04 |
| EmptyState | 2.21 | — | 1.58 | 3.79 |
| ErrorState | 2.18 | — | 1.53 | 3.70 |
| FieldGroup | 2.47 | — | 1.60 | 4.06 |
| FileUpload | 6.30 | — | 2.22 | 8.52 |
| Footer | 1.25 | — | 0.51 | 1.76 |
| Form | 6.41 | — | 2.12 | 8.53 |
| FormField | 2.64 | — | 1.11 | 3.75 |
| Grid | 0.80 | — | 0.38 | 1.18 |
| Header | 16.97 | — | 3.27 | 20.24 |
| Icon | 0.67 | — | 0.33 | 1.00 |
| Input | 2.43 | — | 0.96 | 3.39 |
| InputNumber | 2.82 | — | 0.98 | 3.80 |
| InputTags | 4.39 | — | 1.22 | 5.60 |
| InputTime | 6.10 | — | 0.93 | 7.03 |
| Island | 0.78 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.25 | 0.84 |
| Link | 0.88 | — | 0.32 | 1.20 |
| Listbox | 8.39 | — | 1.49 | 9.88 |
| NavigationMenu | 16.63 | — | 2.93 | 19.56 |
| OverlayPanel | 1.85 | — | 1.49 | 3.34 |
| Page | 1.70 | — | 1.39 | 3.08 |
| Pagination | 2.55 | — | 1.35 | 3.91 |
| PinInput | 2.68 | — | 0.51 | 3.18 |
| Popover | 13.37 | — | 2.45 | 15.82 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.06 | — | 0.97 | 4.03 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.45 | — | 1.29 | 2.74 |
| Select | 19.98 | — | 2.32 | 22.29 |
| Separator | 1.67 | — | 1.48 | 3.15 |
| Sheet | 6.40 | — | 2.17 | 8.56 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.09 | — | 0.99 | 4.08 |
| Spinner | 0.52 | — | 0.53 | 1.05 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.37 | — | 0.99 | 3.36 |
| Stepper | 2.92 | — | 0.98 | 3.90 |
| Switch | 1.96 | — | 1.35 | 3.31 |
| Table | 10.76 | — | 2.70 | 13.46 |
| Tabs | 3.68 | — | 1.23 | 4.92 |
| Textarea | 2.12 | — | 0.52 | 2.64 |
| Timeline | 1.77 | — | 0.90 | 2.67 |
| Toast | 3.19 | — | 1.92 | 5.12 |
| Toaster | 4.22 | — | 2.11 | 6.33 |
| Toggle | 2.30 | — | 1.38 | 3.69 |
| ToggleGroup | 3.45 | — | 1.37 | 4.82 |
| Tooltip | 10.70 | — | 1.63 | 12.32 |
| Tree | 3.94 | — | 0.73 | 4.66 |
| WeftProvider | 0.95 | — | 0.27 | 1.22 |

The five heaviest: DateRangePicker (27.97 KB), Combobox (24.55 KB), Select (22.29 KB), AppShell (22.15 KB), Header (20.24 KB).

## Tree-shaking

`packages/vite/test/tree-shaking.test.ts` builds `import { Button } from
'@weft/ui'` the same way and checks that the output holds no trace of
`Dialog`, `Table`, the overlay stack or the overlays' default icons: a page
pays only for the components it imports, through the index or by path.
`@weft/ui`'s package.json declares its scripts free of side effects and its
stylesheets as having them (`"sideEffects": ["**/*.css"]`), so a bundler
drops every module a page does not use, and the page build, following the
same declaration, inlines only the stylesheets of the components an island
takes from the index.
