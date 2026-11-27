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
| Vue runtime (Vapor) and Vite's preload helper, their own chunk in each build, as in a page build's shared runtime | 17.40 to 27.51 KB |
| `styles/base.css`, inlined on every page by the page build | 0.32 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

85 components. Median: JS 2.93 KB, CSS 1.34 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.76 | 3.56 |
| Alert | 2.85 | 0.62 | 1.72 | 4.57 |
| AppShell | 18.58 | 0.62 | 3.79 | 22.38 |
| Avatar | 1.33 | — | 0.52 | 1.85 |
| AvatarGroup | 1.34 | — | 0.40 | 1.74 |
| Badge | 0.86 | — | 0.80 | 1.66 |
| Banner | 3.05 | 0.63 | 1.57 | 4.62 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.55 | 0.62 | 1.39 | 2.93 |
| Calendar | 11.03 | — | 1.32 | 12.35 |
| Card | 1.63 | 0.62 | 1.37 | 3.00 |
| Checkbox | 1.80 | — | 0.91 | 2.71 |
| CheckboxGroup | 2.37 | — | 1.00 | 3.38 |
| Chip | 1.55 | — | 1.07 | 2.62 |
| CloseButton | 0.62 | — | 0.54 | 1.16 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.28 | — | 2.69 | 24.97 |
| CommandPalette | 11.58 | — | 1.92 | 13.49 |
| ConfirmDialog | 6.67 | 0.62 | 1.96 | 8.63 |
| Container | 0.77 | — | 0.14 | 0.91 |
| ContextMenu | 14.63 | — | 1.90 | 16.53 |
| Cta | 2.70 | 0.62 | 1.79 | 4.49 |
| DataTable | 39.64 | 0.63 | 5.69 | 45.33 |
| DatePicker | 11.83 | 14.83 | 2.72 | 14.55 |
| DateRangePicker | 25.66 | — | 2.73 | 28.39 |
| Dialog | 8.04 | 0.62 | 2.27 | 10.31 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.95 | 0.62 | 2.29 | 9.25 |
| DropdownMenu | 15.08 | 0.62 | 2.34 | 17.42 |
| EmptyState | 2.36 | 0.62 | 1.60 | 3.96 |
| EntityDialog | 31.10 | 0.63 | 4.62 | 35.72 |
| ErrorState | 2.39 | 0.63 | 1.55 | 3.94 |
| Faq | 3.91 | — | 1.11 | 5.02 |
| FeatureGrid | 3.01 | — | 1.26 | 4.28 |
| FieldGroup | 2.65 | 0.62 | 1.63 | 4.27 |
| FileUpload | 6.72 | 0.62 | 2.22 | 8.95 |
| Footer | 1.70 | — | 0.55 | 2.25 |
| Form | 7.11 | 0.62 | 2.13 | 9.24 |
| FormField | 2.93 | — | 1.13 | 4.06 |
| Grid | 1.30 | — | 0.46 | 1.76 |
| Guard | 3.36 | 0.08 | 1.56 | 4.92 |
| Header | 17.12 | 0.62 | 3.38 | 20.49 |
| Hero | 2.68 | 0.62 | 1.67 | 4.35 |
| Icon | 0.67 | — | 0.33 | 1.01 |
| Input | 2.70 | — | 0.97 | 3.67 |
| InputNumber | 3.08 | — | 1.00 | 4.08 |
| InputTags | 4.65 | — | 1.24 | 5.89 |
| InputTime | 6.38 | — | 0.95 | 7.33 |
| Island | 0.77 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.88 | — | 0.32 | 1.20 |
| Listbox | 8.66 | — | 1.54 | 10.20 |
| NavigationMenu | 16.76 | 0.62 | 3.04 | 19.80 |
| OverlayPanel | 1.98 | 0.63 | 1.51 | 3.49 |
| Page | 1.90 | 0.62 | 1.39 | 3.29 |
| Pagination | 3.22 | 0.62 | 1.41 | 4.63 |
| PinInput | 2.94 | — | 0.53 | 3.46 |
| Popover | 13.60 | 0.63 | 2.55 | 16.15 |
| Pricing | 5.97 | 0.62 | 2.23 | 8.20 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.08 | — | 0.99 | 4.07 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.62 | 0.62 | 1.30 | 2.92 |
| Select | 20.29 | — | 2.44 | 22.73 |
| Separator | 1.83 | 0.62 | 1.49 | 3.32 |
| Sheet | 6.59 | 0.62 | 2.27 | 8.87 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.36 | — | 1.02 | 4.38 |
| Spinner | 0.52 | — | 0.53 | 1.05 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.83 | — | 1.06 | 3.88 |
| Stepper | 3.36 | — | 1.04 | 4.40 |
| Switch | 1.99 | — | 1.42 | 3.40 |
| Table | 12.07 | 0.62 | 2.79 | 14.86 |
| Tabs | 4.05 | — | 1.34 | 5.39 |
| Testimonials | 3.15 | — | 1.05 | 4.21 |
| Textarea | 2.39 | — | 0.54 | 2.93 |
| Timeline | 2.25 | — | 0.92 | 3.16 |
| Toast | 3.37 | 0.62 | 1.94 | 5.31 |
| Toaster | 4.40 | 0.62 | 2.12 | 6.52 |
| Toggle | 2.49 | 0.62 | 1.39 | 3.88 |
| ToggleGroup | 3.70 | 0.62 | 1.39 | 5.09 |
| Tooltip | 10.88 | 0.62 | 1.71 | 12.58 |
| Tree | 3.95 | — | 0.74 | 4.69 |
| WeftProvider | 0.97 | — | 0.27 | 1.25 |

The five heaviest: DataTable (45.33 KB), EntityDialog (35.72 KB), DateRangePicker (28.39 KB), Combobox (24.97 KB), Select (22.73 KB).

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
