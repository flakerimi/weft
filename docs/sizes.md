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

84 components. Median: JS 2.88 KB, CSS 1.34 KB.

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
| Checkbox | 1.80 | — | 0.91 | 2.71 |
| CheckboxGroup | 2.37 | — | 1.00 | 3.38 |
| Chip | 1.55 | — | 1.07 | 2.62 |
| CloseButton | 0.62 | — | 0.54 | 1.16 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.31 | — | 2.69 | 25.00 |
| CommandPalette | 11.57 | — | 1.92 | 13.48 |
| ConfirmDialog | 6.51 | — | 1.96 | 8.47 |
| Container | 0.77 | — | 0.14 | 0.91 |
| ContextMenu | 14.62 | — | 1.90 | 16.51 |
| Cta | 2.52 | — | 1.79 | 4.32 |
| DataTable | 40.38 | — | 5.70 | 46.08 |
| DatePicker | 12.38 | 14.87 | 2.72 | 15.11 |
| DateRangePicker | 25.70 | — | 2.73 | 28.43 |
| Dialog | 7.91 | — | 2.27 | 10.19 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.78 | — | 2.29 | 9.07 |
| DropdownMenu | 14.93 | — | 2.34 | 17.27 |
| EmptyState | 2.21 | — | 1.60 | 3.81 |
| EntityDialog | 31.74 | — | 4.63 | 36.37 |
| ErrorState | 2.18 | — | 1.55 | 3.73 |
| Faq | 3.91 | — | 1.11 | 5.02 |
| FeatureGrid | 3.01 | — | 1.26 | 4.28 |
| FieldGroup | 2.49 | — | 1.63 | 4.11 |
| FileUpload | 6.56 | — | 2.23 | 8.79 |
| Footer | 1.70 | — | 0.55 | 2.25 |
| Form | 6.95 | — | 2.13 | 9.08 |
| FormField | 2.92 | — | 1.13 | 4.05 |
| Grid | 1.30 | — | 0.46 | 1.76 |
| Header | 16.94 | — | 3.38 | 20.32 |
| Hero | 2.53 | — | 1.67 | 4.20 |
| Icon | 0.67 | — | 0.33 | 1.01 |
| Input | 2.70 | — | 0.97 | 3.67 |
| InputNumber | 3.08 | — | 1.00 | 4.08 |
| InputTags | 4.65 | — | 1.24 | 5.89 |
| InputTime | 6.38 | — | 0.95 | 7.33 |
| Island | 0.77 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.88 | — | 0.32 | 1.20 |
| Listbox | 8.67 | — | 1.54 | 10.21 |
| NavigationMenu | 16.63 | — | 3.03 | 19.66 |
| OverlayPanel | 1.85 | — | 1.51 | 3.36 |
| Page | 1.70 | — | 1.39 | 3.09 |
| Pagination | 3.08 | — | 1.41 | 4.48 |
| PinInput | 2.94 | — | 0.53 | 3.47 |
| Popover | 13.48 | — | 2.55 | 16.03 |
| Pricing | 5.81 | — | 2.23 | 8.03 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.07 | — | 0.99 | 4.06 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.45 | — | 1.30 | 2.75 |
| Select | 20.26 | — | 2.44 | 22.70 |
| Separator | 1.67 | — | 1.49 | 3.16 |
| Sheet | 6.41 | — | 2.26 | 8.67 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.37 | — | 1.02 | 4.39 |
| Spinner | 0.52 | — | 0.53 | 1.05 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.83 | — | 1.06 | 3.88 |
| Stepper | 3.36 | — | 1.04 | 4.40 |
| Switch | 1.99 | — | 1.42 | 3.41 |
| Table | 11.93 | — | 2.79 | 14.73 |
| Tabs | 4.06 | — | 1.34 | 5.40 |
| Testimonials | 3.15 | — | 1.05 | 4.21 |
| Textarea | 2.39 | — | 0.54 | 2.93 |
| Timeline | 2.25 | — | 0.92 | 3.16 |
| Toast | 3.19 | — | 1.94 | 5.13 |
| Toaster | 4.22 | — | 2.12 | 6.34 |
| Toggle | 2.31 | — | 1.39 | 3.70 |
| ToggleGroup | 3.54 | — | 1.39 | 4.93 |
| Tooltip | 10.71 | — | 1.71 | 12.42 |
| Tree | 3.94 | — | 0.74 | 4.68 |
| WeftProvider | 0.97 | — | 0.27 | 1.25 |

The five heaviest: DataTable (46.08 KB), EntityDialog (36.37 KB), DateRangePicker (28.43 KB), Combobox (25.00 KB), Select (22.70 KB).

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
