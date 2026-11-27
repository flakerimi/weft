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
| Vue runtime (Vapor), its own chunk in each build | 17.40 to 26.94 KB |
| `styles/base.css`, inlined on every page by the page build | 0.32 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

84 components. Median: JS 2.83 KB, CSS 1.34 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.76 | 3.55 |
| Alert | 2.70 | — | 1.72 | 4.42 |
| AppShell | 18.47 | — | 3.79 | 22.25 |
| Avatar | 1.33 | — | 0.52 | 1.85 |
| AvatarGroup | 1.34 | — | 0.40 | 1.74 |
| Badge | 0.86 | — | 0.80 | 1.66 |
| Banner | 2.88 | — | 1.56 | 4.44 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.38 | — | 1.39 | 2.77 |
| Calendar | 11.04 | — | 1.32 | 12.36 |
| Card | 1.45 | — | 1.37 | 2.83 |
| Checkbox | 1.79 | — | 0.91 | 2.70 |
| CheckboxGroup | 2.37 | — | 1.00 | 3.37 |
| Chip | 1.55 | — | 1.07 | 2.62 |
| CloseButton | 0.62 | — | 0.54 | 1.16 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.24 | — | 2.69 | 24.94 |
| CommandPalette | 11.59 | — | 1.92 | 13.50 |
| ConfirmDialog | 6.51 | — | 1.96 | 8.47 |
| Container | 0.77 | — | 0.14 | 0.91 |
| ContextMenu | 14.62 | — | 1.90 | 16.51 |
| Cta | 2.52 | — | 1.79 | 4.32 |
| DataTable | 39.44 | — | 5.70 | 45.15 |
| DatePicker | 12.33 | 14.89 | 2.72 | 15.05 |
| DateRangePicker | 25.62 | — | 2.73 | 28.35 |
| Dialog | 7.81 | — | 2.27 | 10.08 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.78 | — | 2.29 | 9.07 |
| DropdownMenu | 14.93 | — | 2.34 | 17.27 |
| EmptyState | 2.21 | — | 1.60 | 3.81 |
| EntityDialog | 30.83 | — | 4.63 | 35.46 |
| ErrorState | 2.18 | — | 1.55 | 3.73 |
| Faq | 3.91 | — | 1.11 | 5.02 |
| FeatureGrid | 3.01 | — | 1.26 | 4.28 |
| FieldGroup | 2.47 | — | 1.63 | 4.09 |
| FileUpload | 6.48 | — | 2.23 | 8.71 |
| Footer | 1.70 | — | 0.55 | 2.25 |
| Form | 6.83 | — | 2.13 | 8.96 |
| FormField | 2.81 | — | 1.13 | 3.95 |
| Grid | 1.30 | — | 0.46 | 1.76 |
| Header | 16.94 | — | 3.38 | 20.32 |
| Hero | 2.53 | — | 1.67 | 4.20 |
| Icon | 0.67 | — | 0.33 | 1.01 |
| Input | 2.61 | — | 0.97 | 3.58 |
| InputNumber | 3.01 | — | 1.00 | 4.01 |
| InputTags | 4.55 | — | 1.24 | 5.80 |
| InputTime | 6.27 | — | 0.95 | 7.22 |
| Island | 0.77 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.88 | — | 0.32 | 1.20 |
| Listbox | 8.60 | — | 1.54 | 10.14 |
| NavigationMenu | 16.63 | — | 3.03 | 19.66 |
| OverlayPanel | 1.85 | — | 1.51 | 3.36 |
| Page | 1.70 | — | 1.39 | 3.09 |
| Pagination | 3.08 | — | 1.41 | 4.48 |
| PinInput | 2.83 | — | 0.53 | 3.36 |
| Popover | 13.46 | — | 2.55 | 16.01 |
| Pricing | 5.80 | — | 2.23 | 8.02 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.07 | — | 0.99 | 4.06 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.45 | — | 1.30 | 2.75 |
| Select | 20.18 | — | 2.44 | 22.62 |
| Separator | 1.67 | — | 1.49 | 3.16 |
| Sheet | 6.41 | — | 2.26 | 8.67 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.29 | — | 1.02 | 4.31 |
| Spinner | 0.52 | — | 0.53 | 1.05 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.83 | — | 1.06 | 3.88 |
| Stepper | 3.36 | — | 1.04 | 4.40 |
| Switch | 1.97 | — | 1.42 | 3.39 |
| Table | 11.91 | — | 2.79 | 14.71 |
| Tabs | 4.06 | — | 1.34 | 5.40 |
| Testimonials | 3.15 | — | 1.05 | 4.21 |
| Textarea | 2.29 | — | 0.54 | 2.83 |
| Timeline | 2.25 | — | 0.92 | 3.16 |
| Toast | 3.19 | — | 1.94 | 5.13 |
| Toaster | 4.22 | — | 2.12 | 6.34 |
| Toggle | 2.30 | — | 1.39 | 3.69 |
| ToggleGroup | 3.53 | — | 1.39 | 4.91 |
| Tooltip | 10.71 | — | 1.71 | 12.42 |
| Tree | 3.94 | — | 0.74 | 4.68 |
| WeftProvider | 0.97 | — | 0.27 | 1.25 |

The five heaviest: DataTable (45.15 KB), EntityDialog (35.46 KB), DateRangePicker (28.35 KB), Combobox (24.94 KB), Select (22.62 KB).

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
