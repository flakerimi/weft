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
| Vue runtime (Vapor), its own chunk in each build, with Vite's preload helper where the build has one, as a page build's shared runtime has it | 17.67 to 27.84 KB |
| `styles/base.css`, inlined on every page by the page build | 0.32 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

85 components. Median: JS 2.93 KB, CSS 1.34 KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
| Accordion | 2.79 | — | 0.76 | 3.55 |
| Alert | 2.85 | 0.70 | 1.72 | 4.57 |
| AppShell | 18.57 | 0.70 | 3.79 | 22.36 |
| Avatar | 1.33 | — | 0.52 | 1.85 |
| AvatarGroup | 1.35 | — | 0.40 | 1.74 |
| Badge | 0.86 | — | 0.80 | 1.66 |
| Banner | 3.06 | 0.70 | 1.57 | 4.63 |
| Breadcrumb | 1.53 | — | 0.57 | 2.09 |
| Button | 1.54 | 0.69 | 1.39 | 2.93 |
| Calendar | 11.02 | — | 1.32 | 12.34 |
| Card | 1.63 | 0.70 | 1.37 | 3.00 |
| Checkbox | 1.80 | — | 0.91 | 2.71 |
| CheckboxGroup | 2.37 | — | 1.00 | 3.38 |
| Chip | 1.55 | — | 1.07 | 2.62 |
| CloseButton | 0.64 | — | 0.54 | 1.18 |
| Collapsible | 1.33 | — | 0.48 | 1.81 |
| Combobox | 22.36 | — | 2.69 | 25.05 |
| CommandPalette | 11.60 | — | 1.92 | 13.51 |
| ConfirmDialog | 6.70 | 0.70 | 1.96 | 8.66 |
| Container | 0.77 | — | 0.14 | 0.92 |
| ContextMenu | 14.62 | — | 1.90 | 16.52 |
| Cta | 2.70 | 0.69 | 1.79 | 4.49 |
| DataTable | 40.50 | 0.70 | 5.69 | 46.19 |
| DatePicker | 11.83 | 14.84 | 2.72 | 14.55 |
| DateRangePicker | 25.75 | — | 2.73 | 28.47 |
| Dialog | 8.07 | 0.70 | 2.27 | 10.34 |
| Divider | 0.47 | — | 0.34 | 0.81 |
| Drawer | 6.94 | 0.70 | 2.29 | 9.24 |
| DropdownMenu | 15.05 | 0.70 | 2.34 | 17.39 |
| EmptyState | 2.36 | 0.69 | 1.60 | 3.96 |
| EntityDialog | 31.93 | 0.70 | 4.62 | 36.55 |
| ErrorState | 2.37 | 0.70 | 1.55 | 3.92 |
| Faq | 3.91 | — | 1.11 | 5.02 |
| FeatureGrid | 3.02 | — | 1.26 | 4.28 |
| FieldGroup | 2.74 | 0.70 | 1.63 | 4.36 |
| FileUpload | 6.74 | 0.70 | 2.22 | 8.97 |
| Footer | 1.70 | — | 0.55 | 2.25 |
| Form | 7.13 | 0.70 | 2.13 | 9.26 |
| FormField | 2.93 | — | 1.13 | 4.06 |
| Grid | 1.30 | — | 0.46 | 1.76 |
| Guard | 3.66 | 0.08 | 1.72 | 5.38 |
| Header | 17.09 | 0.70 | 3.38 | 20.47 |
| Hero | 2.70 | 0.70 | 1.67 | 4.37 |
| Icon | 0.67 | — | 0.33 | 1.01 |
| Input | 2.72 | — | 0.97 | 3.69 |
| InputNumber | 3.10 | — | 1.00 | 4.10 |
| InputTags | 4.67 | — | 1.24 | 5.91 |
| InputTime | 6.42 | — | 0.95 | 7.37 |
| Island | 0.77 | — | 0.25 | 1.03 |
| Kbd | 0.59 | — | 0.27 | 0.86 |
| Link | 0.87 | — | 0.32 | 1.20 |
| Listbox | 8.67 | — | 1.54 | 10.21 |
| NavigationMenu | 16.78 | 0.70 | 3.04 | 19.82 |
| OverlayPanel | 1.98 | 0.70 | 1.51 | 3.49 |
| Page | 1.90 | 0.70 | 1.39 | 3.29 |
| Pagination | 3.22 | 0.70 | 1.41 | 4.62 |
| PinInput | 2.94 | — | 0.53 | 3.47 |
| Popover | 13.64 | 0.70 | 2.55 | 16.19 |
| Pricing | 5.98 | 0.70 | 2.23 | 8.20 |
| Progress | 0.78 | — | 0.85 | 1.63 |
| RadioGroup | 3.08 | — | 0.99 | 4.07 |
| ScrollArea | 1.02 | — | 0.56 | 1.58 |
| Section | 1.62 | 0.70 | 1.30 | 2.92 |
| Select | 20.26 | — | 2.44 | 22.70 |
| Separator | 1.83 | 0.69 | 1.49 | 3.32 |
| Sheet | 6.60 | 0.70 | 2.27 | 8.87 |
| Skeleton | 0.53 | — | 0.45 | 0.97 |
| Slider | 3.38 | — | 1.02 | 4.39 |
| Spinner | 0.51 | — | 0.53 | 1.04 |
| Stack | 0.35 | — | 0.27 | 0.62 |
| StatCard | 2.83 | — | 1.06 | 3.88 |
| Stepper | 3.36 | — | 1.04 | 4.40 |
| Switch | 1.99 | — | 1.42 | 3.40 |
| Table | 12.08 | 0.70 | 2.79 | 14.88 |
| Tabs | 4.05 | — | 1.34 | 5.39 |
| Testimonials | 3.15 | — | 1.05 | 4.20 |
| Textarea | 2.41 | — | 0.54 | 2.95 |
| Timeline | 2.25 | — | 0.92 | 3.17 |
| Toast | 3.37 | 0.70 | 1.94 | 5.31 |
| Toaster | 4.40 | 0.70 | 2.12 | 6.52 |
| Toggle | 2.48 | 0.70 | 1.39 | 3.86 |
| ToggleGroup | 3.70 | 0.70 | 1.39 | 5.09 |
| Tooltip | 10.87 | 0.69 | 1.71 | 12.58 |
| Tree | 3.95 | — | 0.74 | 4.69 |
| WeftProvider | 0.98 | — | 0.27 | 1.25 |

The five heaviest: DataTable (46.19 KB), EntityDialog (36.55 KB), DateRangePicker (28.47 KB), Combobox (25.05 KB), Select (22.70 KB).

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
