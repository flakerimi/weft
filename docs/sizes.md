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
| Vue runtime (Vapor), its own chunk in each build | 17.67 to 27.18 KB |
| `styles/base.css`, inlined on every page by the page build | 0.32 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

84 components. Median: JS 2.88 KB, CSS 1.34 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.76 | 3.55 |
| Alert | 2.70 | — | 1.72 | 4.42 |
| AppShell | 18.47 | — | 3.79 | 22.26 |
| Avatar | 1.33 | — | 0.52 | 1.85 |
| AvatarGroup | 1.35 | — | 0.40 | 1.74 |
| Badge | 0.86 | — | 0.80 | 1.66 |
| Banner | 2.88 | — | 1.56 | 4.44 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.39 | — | 1.39 | 2.78 |
| Calendar | 11.03 | — | 1.32 | 12.35 |
| Card | 1.45 | — | 1.37 | 2.83 |
| Checkbox | 1.80 | — | 0.91 | 2.71 |
| CheckboxGroup | 2.37 | — | 1.00 | 3.38 |
| Chip | 1.55 | — | 1.07 | 2.62 |
| CloseButton | 0.64 | — | 0.54 | 1.18 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.28 | — | 2.69 | 24.97 |
| CommandPalette | 11.59 | — | 1.92 | 13.51 |
| ConfirmDialog | 6.51 | — | 1.96 | 8.47 |
| Container | 0.77 | — | 0.14 | 0.92 |
| ContextMenu | 14.60 | — | 1.90 | 16.50 |
| Cta | 2.52 | — | 1.79 | 4.31 |
| DataTable | 39.57 | — | 5.70 | 45.27 |
| DatePicker | 12.39 | 14.84 | 2.72 | 15.12 |
| DateRangePicker | 25.73 | — | 2.73 | 28.46 |
| Dialog | 7.91 | — | 2.27 | 10.19 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.78 | — | 2.29 | 9.07 |
| DropdownMenu | 14.92 | — | 2.34 | 17.26 |
| EmptyState | 2.21 | — | 1.60 | 3.81 |
| EntityDialog | 30.95 | — | 4.63 | 35.58 |
| ErrorState | 2.18 | — | 1.55 | 3.73 |
| Faq | 3.91 | — | 1.11 | 5.02 |
| FeatureGrid | 3.02 | — | 1.26 | 4.28 |
| FieldGroup | 2.58 | — | 1.63 | 4.20 |
| FileUpload | 6.59 | — | 2.23 | 8.82 |
| Footer | 1.70 | — | 0.55 | 2.25 |
| Form | 6.99 | — | 2.13 | 9.12 |
| FormField | 2.93 | — | 1.13 | 4.06 |
| Grid | 1.30 | — | 0.46 | 1.76 |
| Header | 16.95 | — | 3.38 | 20.32 |
| Hero | 2.53 | — | 1.67 | 4.20 |
| Icon | 0.67 | — | 0.33 | 1.01 |
| Input | 2.71 | — | 0.97 | 3.68 |
| InputNumber | 3.10 | — | 1.00 | 4.10 |
| InputTags | 4.66 | — | 1.24 | 5.90 |
| InputTime | 6.42 | — | 0.95 | 7.37 |
| Island | 0.77 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.87 | — | 0.32 | 1.20 |
| Listbox | 8.67 | — | 1.54 | 10.21 |
| NavigationMenu | 16.63 | — | 3.03 | 19.66 |
| OverlayPanel | 1.85 | — | 1.51 | 3.36 |
| Page | 1.70 | — | 1.39 | 3.09 |
| Pagination | 3.08 | — | 1.41 | 4.48 |
| PinInput | 2.94 | — | 0.53 | 3.47 |
| Popover | 13.46 | — | 2.55 | 16.01 |
| Pricing | 5.80 | — | 2.23 | 8.02 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.07 | — | 0.99 | 4.06 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.45 | — | 1.30 | 2.75 |
| Select | 20.28 | — | 2.44 | 22.72 |
| Separator | 1.67 | — | 1.49 | 3.16 |
| Sheet | 6.41 | — | 2.26 | 8.67 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.37 | — | 1.02 | 4.39 |
| Spinner | 0.51 | — | 0.53 | 1.04 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.83 | — | 1.06 | 3.88 |
| Stepper | 3.36 | — | 1.04 | 4.40 |
| Switch | 2.00 | — | 1.42 | 3.41 |
| Table | 11.93 | — | 2.79 | 14.72 |
| Tabs | 4.06 | — | 1.34 | 5.40 |
| Testimonials | 3.15 | — | 1.05 | 4.20 |
| Textarea | 2.41 | — | 0.54 | 2.95 |
| Timeline | 2.25 | — | 0.92 | 3.17 |
| Toast | 3.19 | — | 1.94 | 5.13 |
| Toaster | 4.23 | — | 2.12 | 6.35 |
| Toggle | 2.31 | — | 1.39 | 3.70 |
| ToggleGroup | 3.54 | — | 1.39 | 4.92 |
| Tooltip | 10.68 | — | 1.71 | 12.39 |
| Tree | 3.95 | — | 0.74 | 4.69 |
| WeftProvider | 0.98 | — | 0.27 | 1.25 |

The five heaviest: DataTable (45.27 KB), EntityDialog (35.58 KB), DateRangePicker (28.46 KB), Combobox (24.97 KB), Select (22.72 KB).

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
