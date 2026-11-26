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
| `styles/base.css`, inlined on every page by the page build | 0.32 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

76 components. Median: JS 2.61 KB, CSS 1.30 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.76 | 3.55 |
| Alert | 2.70 | — | 1.72 | 4.42 |
| AppShell | 18.50 | — | 3.79 | 22.29 |
| Avatar | 1.33 | — | 0.53 | 1.86 |
| AvatarGroup | 1.34 | — | 0.41 | 1.75 |
| Badge | 0.87 | — | 0.80 | 1.67 |
| Banner | 2.88 | — | 1.56 | 4.44 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.38 | — | 1.39 | 2.77 |
| Calendar | 11.04 | — | 1.31 | 12.35 |
| Card | 1.45 | — | 1.37 | 2.83 |
| Checkbox | 1.79 | — | 0.91 | 2.70 |
| CheckboxGroup | 2.36 | — | 1.00 | 3.36 |
| Chip | 1.55 | — | 1.08 | 2.63 |
| CloseButton | 0.62 | — | 0.54 | 1.16 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.07 | — | 2.67 | 24.75 |
| CommandPalette | 11.55 | — | 1.90 | 13.45 |
| ConfirmDialog | 6.55 | — | 1.96 | 8.51 |
| Container | 0.77 | — | 0.14 | 0.91 |
| ContextMenu | 14.59 | — | 1.88 | 16.48 |
| DatePicker | 12.17 | 14.87 | 2.71 | 14.88 |
| DateRangePicker | 25.45 | — | 2.72 | 28.18 |
| Dialog | 7.69 | — | 2.27 | 9.97 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.78 | — | 2.29 | 9.08 |
| DropdownMenu | 14.88 | — | 2.32 | 17.21 |
| EmptyState | 2.21 | — | 1.59 | 3.80 |
| ErrorState | 2.18 | — | 1.54 | 3.72 |
| FieldGroup | 2.47 | — | 1.63 | 4.09 |
| FileUpload | 6.32 | — | 2.23 | 8.55 |
| Footer | 1.29 | — | 0.55 | 1.84 |
| Form | 6.65 | — | 2.13 | 8.78 |
| FormField | 2.64 | — | 1.13 | 3.77 |
| Grid | 0.86 | — | 0.41 | 1.26 |
| Header | 16.98 | — | 3.38 | 20.35 |
| Icon | 0.67 | — | 0.33 | 1.00 |
| Input | 2.43 | — | 0.97 | 3.40 |
| InputNumber | 2.83 | — | 1.00 | 3.83 |
| InputTags | 4.39 | — | 1.24 | 5.63 |
| InputTime | 6.10 | — | 0.95 | 7.06 |
| Island | 0.78 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.88 | — | 0.32 | 1.20 |
| Listbox | 8.41 | — | 1.51 | 9.92 |
| NavigationMenu | 16.67 | — | 3.04 | 19.71 |
| OverlayPanel | 1.85 | — | 1.51 | 3.36 |
| Page | 1.70 | — | 1.39 | 3.09 |
| Pagination | 2.61 | — | 1.36 | 3.97 |
| PinInput | 2.68 | — | 0.53 | 3.21 |
| Popover | 13.43 | — | 2.55 | 15.98 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.06 | — | 0.99 | 4.05 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.45 | — | 1.30 | 2.75 |
| Select | 20.05 | — | 2.42 | 22.47 |
| Separator | 1.67 | — | 1.49 | 3.16 |
| Sheet | 6.41 | — | 2.26 | 8.67 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.09 | — | 1.02 | 4.11 |
| Spinner | 0.52 | — | 0.53 | 1.05 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.39 | — | 1.01 | 3.40 |
| Stepper | 2.97 | — | 1.00 | 3.97 |
| Switch | 1.96 | — | 1.42 | 3.38 |
| Table | 11.11 | — | 2.75 | 13.86 |
| Tabs | 3.74 | — | 1.28 | 5.02 |
| Textarea | 2.12 | — | 0.54 | 2.66 |
| Timeline | 1.83 | — | 0.90 | 2.73 |
| Toast | 3.19 | — | 1.94 | 5.13 |
| Toaster | 4.22 | — | 2.12 | 6.34 |
| Toggle | 2.30 | — | 1.39 | 3.69 |
| ToggleGroup | 3.45 | — | 1.39 | 4.84 |
| Tooltip | 10.70 | — | 1.71 | 12.41 |
| Tree | 3.95 | — | 0.74 | 4.69 |
| WeftProvider | 0.97 | — | 0.27 | 1.25 |

The five heaviest: DateRangePicker (28.18 KB), Combobox (24.75 KB), Select (22.47 KB), AppShell (22.29 KB), Header (20.35 KB).

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
