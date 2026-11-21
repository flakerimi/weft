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
| Vue runtime (Vapor), its own chunk in each build | 17.40 to 26.25 KB |
| `styles/base.css`, inlined on every page by the page build | 0.31 KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

76 components. Median: JS 2.69 KB, CSS 1.22 KB.

| Component | JS | CSS | Total |
| --- | ---: | ---: | ---: |
| Accordion | 2.92 | 0.74 | 3.66 |
| Alert | 2.89 | 1.64 | 4.53 |
| AppShell | 18.26 | 3.55 | 21.81 |
| Avatar | 1.46 | 0.53 | 1.99 |
| AvatarGroup | 1.47 | 0.41 | 1.88 |
| Badge | 1.00 | 0.73 | 1.72 |
| Banner | 2.86 | 1.48 | 4.34 |
| Breadcrumb | 1.63 | 0.57 | 2.19 |
| Button | 1.57 | 1.31 | 2.88 |
| Calendar | 10.92 | 1.18 | 12.10 |
| Card | 1.63 | 1.28 | 2.91 |
| Checkbox | 1.79 | 0.83 | 2.62 |
| CheckboxGroup | 2.35 | 0.92 | 3.27 |
| Chip | 1.67 | 1.01 | 2.68 |
| CloseButton | 0.62 | 0.52 | 1.14 |
| Collapsible | 1.33 | 0.48 | 1.81 |
| Combobox | 21.99 | 2.56 | 24.55 |
| CommandPalette | 11.56 | 1.86 | 13.42 |
| ConfirmDialog | 6.60 | 1.80 | 8.40 |
| Container | 0.66 | 0.14 | 0.80 |
| ContextMenu | 14.57 | 1.68 | 16.25 |
| DatePicker | 24.84 | 2.50 | 27.33 |
| DateRangePicker | 25.37 | 2.50 | 27.87 |
| Dialog | 7.76 | 2.10 | 9.86 |
| Divider | 0.47 | 0.33 | 0.80 |
| Drawer | 6.84 | 2.14 | 8.98 |
| DropdownMenu | 14.87 | 2.14 | 17.01 |
| EmptyState | 2.34 | 1.50 | 3.84 |
| ErrorState | 2.37 | 1.46 | 3.82 |
| FieldGroup | 2.66 | 1.53 | 4.19 |
| FileUpload | 6.42 | 2.14 | 8.57 |
| Footer | 1.23 | 0.51 | 1.74 |
| Form | 6.53 | 2.04 | 8.57 |
| FormField | 2.78 | 1.11 | 3.89 |
| Grid | 0.80 | 0.38 | 1.18 |
| Header | 16.89 | 3.19 | 20.08 |
| Icon | 0.80 | 0.33 | 1.13 |
| Input | 2.57 | 0.96 | 3.53 |
| InputNumber | 2.82 | 0.98 | 3.81 |
| InputTags | 4.51 | 1.22 | 5.72 |
| InputTime | 6.16 | 0.93 | 7.09 |
| Island | 0.64 | 0.25 | 0.90 |
| Kbd | 0.59 | 0.25 | 0.84 |
| Link | 0.86 | 0.32 | 1.18 |
| Listbox | 8.43 | 1.49 | 9.92 |
| NavigationMenu | 16.53 | 2.85 | 19.37 |
| OverlayPanel | 2.02 | 1.42 | 3.44 |
| Page | 1.80 | 1.31 | 3.11 |
| Pagination | 2.71 | 1.28 | 4.00 |
| PinInput | 2.69 | 0.51 | 3.19 |
| Popover | 13.49 | 2.37 | 15.86 |
| Progress | 0.78 | 0.78 | 1.56 |
| RadioGroup | 3.06 | 0.90 | 3.97 |
| ScrollArea | 1.02 | 0.56 | 1.58 |
| Section | 1.63 | 1.23 | 2.85 |
| Select | 19.95 | 2.32 | 22.27 |
| Separator | 1.84 | 1.41 | 3.25 |
| Sheet | 6.45 | 2.09 | 8.54 |
| Skeleton | 0.53 | 0.45 | 0.97 |
| Slider | 3.09 | 0.92 | 4.01 |
| Spinner | 0.52 | 0.53 | 1.05 |
| Stack | 0.35 | 0.27 | 0.62 |
| StatCard | 2.43 | 0.92 | 3.35 |
| Stepper | 3.05 | 0.91 | 3.97 |
| Switch | 1.98 | 1.28 | 3.25 |
| Table | 10.69 | 2.64 | 13.32 |
| Tabs | 3.81 | 1.17 | 4.98 |
| Textarea | 2.12 | 0.52 | 2.63 |
| Timeline | 1.90 | 0.83 | 2.72 |
| Toast | 3.35 | 1.85 | 5.20 |
| Toaster | 4.42 | 2.03 | 6.44 |
| Toggle | 2.45 | 1.31 | 3.76 |
| ToggleGroup | 3.64 | 1.30 | 4.94 |
| Tooltip | 10.74 | 1.55 | 12.29 |
| Tree | 3.95 | 0.73 | 4.68 |
| WeftProvider | 0.80 | 0.27 | 1.08 |

The five heaviest: DateRangePicker (27.87 KB), DatePicker (27.33 KB), Combobox (24.55 KB), Select (22.27 KB), AppShell (21.81 KB).

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
