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

## Shared by every page

| What | Brotli |
| --- | ---: |
| Vue runtime (Vapor), its own chunk in each build | 17.40 to 26.23 KB |
| `styles/base.css`, inlined on every page by the page build | 0.31 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

76 components. Median: JS 2.69 KB, CSS 1.29 KB.

| Component | JS | CSS | Total |
| --- | ---: | ---: | ---: |
| Accordion | 2.92 | 0.74 | 3.66 |
| Alert | 2.88 | 1.71 | 4.59 |
| AppShell | 18.57 | 3.68 | 22.25 |
| Avatar | 1.46 | 0.53 | 1.99 |
| AvatarGroup | 1.47 | 0.41 | 1.88 |
| Badge | 1.00 | 0.79 | 1.79 |
| Banner | 3.03 | 1.56 | 4.58 |
| Breadcrumb | 1.65 | 0.57 | 2.22 |
| Button | 1.55 | 1.38 | 2.94 |
| Calendar | 10.99 | 1.31 | 12.30 |
| Card | 1.63 | 1.35 | 2.98 |
| Checkbox | 1.79 | 0.89 | 2.68 |
| CheckboxGroup | 2.35 | 0.99 | 3.33 |
| Chip | 1.67 | 1.08 | 2.75 |
| CloseButton | 0.62 | 0.52 | 1.14 |
| Collapsible | 1.33 | 0.48 | 1.81 |
| Combobox | 22.09 | 2.56 | 24.66 |
| CommandPalette | 11.66 | 1.86 | 13.53 |
| ConfirmDialog | 6.66 | 1.88 | 8.53 |
| Container | 0.76 | 0.14 | 0.90 |
| ContextMenu | 14.58 | 1.77 | 16.35 |
| DatePicker | 24.95 | 2.62 | 27.57 |
| DateRangePicker | 25.41 | 2.62 | 28.03 |
| Dialog | 7.80 | 2.18 | 9.98 |
| Divider | 0.47 | 0.33 | 0.80 |
| Drawer | 6.91 | 2.21 | 9.12 |
| DropdownMenu | 14.94 | 2.21 | 17.15 |
| EmptyState | 2.33 | 1.58 | 3.91 |
| ErrorState | 2.36 | 1.53 | 3.89 |
| FieldGroup | 2.66 | 1.60 | 4.26 |
| FileUpload | 6.46 | 2.22 | 8.68 |
| Footer | 1.25 | 0.51 | 1.76 |
| Form | 6.57 | 2.12 | 8.69 |
| FormField | 2.78 | 1.11 | 3.89 |
| Grid | 0.80 | 0.38 | 1.18 |
| Header | 17.12 | 3.27 | 20.39 |
| Icon | 0.80 | 0.33 | 1.13 |
| Input | 2.57 | 0.96 | 3.53 |
| InputNumber | 2.82 | 0.98 | 3.81 |
| InputTags | 4.52 | 1.22 | 5.73 |
| InputTime | 6.22 | 0.93 | 7.15 |
| Island | 0.78 | 0.25 | 1.03 |
| Kbd | 0.59 | 0.25 | 0.84 |
| Link | 0.88 | 0.32 | 1.20 |
| Listbox | 8.54 | 1.49 | 10.03 |
| NavigationMenu | 16.73 | 2.93 | 19.67 |
| OverlayPanel | 2.01 | 1.49 | 3.50 |
| Page | 1.87 | 1.39 | 3.26 |
| Pagination | 2.71 | 1.35 | 4.07 |
| PinInput | 2.69 | 0.51 | 3.19 |
| Popover | 13.49 | 2.45 | 15.93 |
| Progress | 0.78 | 0.85 | 1.63 |
| RadioGroup | 3.06 | 0.97 | 4.03 |
| ScrollArea | 1.02 | 0.56 | 1.58 |
| Section | 1.61 | 1.29 | 2.90 |
| Select | 20.11 | 2.32 | 22.42 |
| Separator | 1.84 | 1.48 | 3.32 |
| Sheet | 6.54 | 2.17 | 8.70 |
| Skeleton | 0.53 | 0.45 | 0.97 |
| Slider | 3.09 | 0.99 | 4.08 |
| Spinner | 0.52 | 0.53 | 1.05 |
| Stack | 0.35 | 0.27 | 0.62 |
| StatCard | 2.50 | 0.99 | 3.49 |
| Stepper | 3.05 | 0.98 | 4.04 |
| Switch | 1.98 | 1.35 | 3.32 |
| Table | 10.95 | 2.70 | 13.65 |
| Tabs | 3.81 | 1.23 | 5.04 |
| Textarea | 2.12 | 0.52 | 2.63 |
| Timeline | 1.90 | 0.90 | 2.79 |
| Toast | 3.35 | 1.92 | 5.27 |
| Toaster | 4.41 | 2.11 | 6.51 |
| Toggle | 2.46 | 1.38 | 3.84 |
| ToggleGroup | 3.63 | 1.37 | 5.00 |
| Tooltip | 10.83 | 1.63 | 12.45 |
| Tree | 4.05 | 0.73 | 4.77 |
| WeftProvider | 0.95 | 0.27 | 1.22 |

The five heaviest: DateRangePicker (28.03 KB), DatePicker (27.57 KB), Combobox (24.66 KB), Select (22.42 KB), AppShell (22.25 KB).

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
