# Weft: design

Date: 2026-11-20
Repository: `github.com/flakerimi/weft` (to be created). Local folder: `/Volumes/External/UnifiedUI`.

## Intent

The team builds many products: landing pages, admin panels such as Negenet,
dashboards with live network graphs. Backends are written in Go. Frontends have
been Nuxt with Nuxt UI, and every app ends up rebuilding the same higher-level
pieces by hand (Negenet's admin alone has four different stat-card components,
per-entity add/edit/delete modals, its own table, pagination, sidebar and
confirm dialog) while sitting on a different Nuxt UI version.

Weft is the team's own UI system, to be open-sourced. It gives:

- **Full control.** Every component, its API and its look belong to Weft. Third
  parties may sit underneath, never in the public API.
- **One system that adapts.** A component is written once and chooses its
  interaction from how it is being used (mouse or touch, landing or app). There
  are no `Mobile*` variants.
- **A Go-first app shape.** A new product is one Go binary that serves its API
  and embeds its Vue frontend.
- **Built for agents.** Coding agents assemble screens from large, strict
  building blocks instead of hand-writing divs, and can check their own work.

Success: a new product (landing plus admin plus live dashboard) is created with
`weft create`, built by an agent mostly from Weft blocks, and shipped as one
binary, with screens that look like they belong together.

## Goals

1. A Vue 3 component library, `@weft/ui`, with clean neutral styling and
   theming through design tokens.
2. Adaptive behaviour on two independent axes (below), inside the components.
3. Higher-level building blocks, not just primitives: app shell, data table with
   CRUD actions and pagination, entity form dialog, confirm dialog, stat card,
   multi-step wizard, charts, landing sections.
4. A Go module that embeds the built Vue app and serves it with the API, plus
   helpers for live data (server-sent events) that feed Vue reactivity.
5. A `weft` CLI (Go) that creates, runs, builds and checks apps.
6. An agent kit: a machine-readable component catalogue, Claude Code skills, an
   MCP server exposing the catalogue, and a check command that rejects
   off-system markup.
7. Open source from the start, with a README worth reading.

## Non-goals (for now)

- Migrating the existing Nuxt apps. Weft targets new apps only.
- A Nuxt module. Possible later; not built now.
- Server-side rendering per request. Apps are client-rendered and served by Go.
- Native mobile apps. "Mobile" means a phone browser.
- Supporting Svelte, Solid or React.

## Anti-goals (would make it a failure even if it works)

- A wrapper over Nuxt UI or any other styled kit.
- Floating UI, Unovis or Tailwind names leaking into app code, so that
  replacing them breaks apps.
- A virtual-DOM component library inside Weft, which would force every app to
  ship the virtual-DOM runtime.
- More than one way to build the same thing: prop names that differ between
  components, free-form class soup as the normal way to style a screen.
- Separate mobile components the developer or agent has to remember to pick.
- Go rendering HTML templates with Vue sprinkled on top (two template systems,
  every component written twice).

## Constraints

- Vue 3.6 with Vapor mode, starting on the release candidate (`3.6.0-rc.16`,
  published 2026-11-20; stable is `3.5.46`). Vapor has been feature-complete
  since RC 1 (July 2026), but RCs still land a dozen Vapor fixes a week. Pin
  the exact RC, move up deliberately, and switch to `3.6.x` stable when it
  ships. Sources: Vue's 3.6 release notes and the `minor`-branch CHANGELOG.
- Go 1.26 (installed locally: `go1.26.2`).
- Versions checked on npm on 2026-11-20: `vite 8.3.8`,
  `@vitejs/plugin-vue 6.0.11`, `vue-router 5.5.1`, `tailwindcss 4.3.5`,
  `@floating-ui/dom 1.8.0`, `@internationalized/date 3.12.4`,
  `@unovis/ts 1.8.1`, `@vueuse/core 15.0.0`, `vitest 5.1.1`,
  `typescript 7.0.2`, `vue-tsc 3.3.14`, `pnpm 12.21.0`,
  `@modelcontextprotocol/sdk 1.32.2`. Re-check with `npm view <pkg> version`
  before adding any dependency, and confirm each one works in Vapor
  components (VueUse and the router in particular).
- Names: GitHub `flakerimi/weft` is free; no `weft` binary on this machine.
  Package names `@weft/*` are used inside the workspace and the apps only,
  never published.
- Hosting: `weft.base.al` (all of `*.base.al`) points at the owner's Lisa
  server `lisa.base.al`. The site, install script and catalogue are pushed
  there with `lisa server push --name weft`.

## Approaches considered

For the component library:

1. **Wrap Nuxt UI.** Fastest, but keeps the dependency on their releases and
   API. Rejected: it is the opposite of full control.
2. **Own components on a hidden headless core (Reka UI), virtual DOM now,
   Vapor later.** Fastest start on stable Vue 3.5. Rejected after research:
   Reka is virtual-DOM based with no Vapor work on its v3 roadmap, Vue warns
   that mixing modes is rough (Nuxt UI users report runtime errors), so Reka
   would have to be replaced before Weft could ever be pure Vapor.
3. **Vapor-first, Weft's own headless layer. (Chosen.)** Weft writes every
   component, API and style, and its own focus, keyboard, dismiss and
   roving-focus behaviour as composables. Framework-free helpers do the
   hardest mechanics: `@floating-ui/dom` for positioning,
   `@internationalized/date` for calendars and locales, `@unovis/ts` (the
   plain-JS build) inside chart components. Apps are created with
   `createVaporApp` and ship no virtual-DOM runtime. Cost: weeks of
   accessibility work up front and following the RC.
4. **Everything from scratch, no helpers at all.** Rejected: positioning and
   date maths are solved problems with no Vue in them.

For the app framework:

1. **Nuxt.** Mature, but its value is mostly in the Node server, which the team
   would replace with Go anyway.
2. **Astro / Svelte / Solid.** Switching language means rewriting everything and
   agents are less reliable in them; Astro solves content sites, not apps.
3. **Go host embedding a Vite-built Vue app, with a thin Weft app layer.
   (Chosen.)** One binary, no Node in production, reactivity stays in the
   browser. Without a server renderer, the remaining framework parts (file
   routes, layouts, auto-imports) are small enough to own.

## Architecture

```
weft create myapp
myapp/
  main.go            Go server: API routes + weft.Serve(embedded frontend)
  api/               the product's Go handlers
  web/               the Vue app (Vite)
    pages/           file-based routes
    layouts/         landing, app, ... (declare the surface)
    components/      product-specific components
  weft.config.ts     theme, app name, options
  go.mod, package.json
```

Production: `weft build` runs the Vite build into `web/dist`, then `go build`
embeds it with `//go:embed`. The binary serves `/api/*` from Go and everything
else from the embedded files, with a fallback to `index.html` for client routes.

Development: `weft dev` starts the Vite dev server and the Go server together,
with Vite proxying `/api` to Go, and restarts Go on `.go` changes.

### Repository layout (monorepo)

```
weft/
  go.mod                     module github.com/flakerimi/weft
  cmd/weft/                  the CLI
  server/                    Go package: Serve, SPA fallback, SSE helpers
  templates/                 what `weft create` writes
  packages/
    ui/                      @weft/ui        components, composables, tokens
    vite/                    @weft/vite      routes, layouts, auto-imports
    themes/                  @weft/themes    neutral (default), lisa
    eslint-plugin/           @weft/eslint-plugin
  catalog/                   generated component catalogue (JSON)
  skills/                    Claude Code skills for building with Weft
  playground/                a Go + Vue app that shows every component
  docs/
```

pnpm workspace for the JavaScript packages; one Go module for server and CLI.

## The adaptive model

Two independent axes, both readable through one composable, `useWeft()`.

| Axis | Values | Source |
|---|---|---|
| **input** | `pointer`, `touch` | browser media queries `(pointer: coarse)` and `(hover: hover)`, updated live |
| **surface** | `landing`, `app` | declared by the layout the page uses |

A third, derived value, **width** (`narrow`, `regular`, `wide`), comes from the
width of the component's container, not the window, so a component in a side
panel behaves narrow on a big screen.

Rules:

- Detect capability, never device names.
- Components adapt internally. Each component's docs state what it does per
  axis. Examples: `DatePicker` is a calendar popover with typed input on
  `pointer`, a bottom sheet with large targets on `touch`. `FileUpload` is a drop
  zone with a browse fallback on `pointer`, one button offering camera, photos or
  files on `touch`. `Select` and `Menu` become sheets on touch, `Dialog` becomes
  full-screen on `narrow`, `Tooltip` is not shown on touch, `DataTable` becomes
  stacked cards on `narrow`.
- `surface` changes density, type scale and spacing tokens, not behaviour.
- `<WeftProvider input="touch" surface="app">` overrides either axis for a
  subtree (testing, kiosks, previews).

## Components

Plain names, no prefix: `<Button>`, `<DatePicker>`, `<DataTable>`. They must be
written PascalCase in templates; lowercase `<button>` stays the native element.
`@weft/vite` auto-imports them; plain imports from `@weft/ui` also work.

API conventions, enforced across every component:

- `v-model` for the main value, always.
- The same prop names everywhere: `size` (`sm`, `md`, `lg`), `tone`
  (`neutral`, `primary`, `danger`, `warning`, `success`), `variant` only where a
  component genuinely has visual variants, `loading`, `disabled`.
- Icons by name string (`icon="trash"`) from one bundled icon set.
- Slots named the same across components (`header`, `footer`, `actions`, `empty`).
- `class` is accepted but the catalogue and skills steer away from it; layout
  is done with Weft's `Stack`, `Grid`, `Page`, `Section` components.

Component list by tier (see Phases for what each release contains):

1. **Layout:** `Stack`, `Grid`, `Page`, `Section`, `Card`, `Divider`.
2. **Primitives:** `Button`, `Input`, `Textarea`, `Select`, `Combobox`,
   `Checkbox`, `Switch`, `Radio`, `DatePicker`, `FileUpload`, `Dialog`, `Sheet`,
   `Menu`, `Tooltip`, `Tabs`, `Badge`, `Avatar`, `Toast`, `Form` with field
   validation.
3. **App blocks:** `AppShell` (sidebar on wide, bottom navigation on narrow
   touch), `DataTable` (sorting, filtering, pagination, row actions, selection),
   `EntityDialog` (create and edit from a field schema), `ConfirmDialog`,
   `StatCard`, `Wizard`, `EmptyState`, `ErrorState`.
4. **Charts** (`@weft/ui/charts`, `@unovis/ts` inside, optional): `LineChart`,
   `AreaChart`, `BarChart`, `DonutChart`, `Sparkline`, all accepting live data.
5. **Landing blocks:** `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`,
   `Cta`, `Footer`.

### Theming

Design tokens are CSS custom properties (colour, radius, spacing, type scale,
motion), with a light and dark set. A theme is one token file.
`@weft/themes` ships `neutral` (default) and `lisa`. The `lisa` theme is
generated from Lisa's token contract, `Lisa/docs/contract/tokens.json` (the
same source as `Lisa/frameworks/ui/lib/tokens.dart`), by a small script in
`packages/themes`, so web and OS stay in step. (Not from `lib/src/colors.dart`
and friends: those are forks of Flutter's Material defaults.)

Components are styled with Tailwind v4 internally, mapped to the tokens, and
ship compiled CSS, so apps do not need Tailwind. Apps may still use Tailwind
for their own components; `weft create` sets it up with the same tokens.

### Vapor

- Every Weft component is `<script setup vapor>`. No Options API,
  `getCurrentInstance`, `app.config.globalProperties`, `v-memo` or render
  functions returning VNodes.
- Created apps mount with `createVaporApp` and do not install
  `vaporInteropPlugin`. A dependency that needs the virtual DOM is not added;
  `weft check` flags it.
- Accessibility behaviour lives in `packages/ui/src/headless/` as composables
  (`useFocusTrap`, `useDismiss`, `useRovingFocus`, `useFloating`,
  `useTypeahead`, `useId`), following the WAI-ARIA Authoring Practices
  patterns, each with keyboard and screen-reader tests.
- Safety net: components remain valid ordinary SFCs. If Vapor slips, a build
  flag strips the `vapor` attribute and ships a virtual-DOM build of the same
  source. CI builds both and runs the tests on both, so the fallback never rots.

## Go side

Package `github.com/flakerimi/weft/server`:

- `weft.Serve(dist fs.FS, api http.Handler) http.Handler`: API under `/api`,
  static files with cache headers, `index.html` fallback for client routes.
- `weft.SSE(w, r)`: a small server-sent-events writer (send typed events,
  heartbeat, close on client disconnect).
- Uses only the standard library `net/http`, so it works with any router.

Vue side, in `@weft/ui`:

- `useLive(url)` subscribes to an SSE stream and returns reactive state, with
  reconnect and back-off; each event replaces the state (status cards).
  `useLiveSeries(url, { window })` appends each event as a point and keeps a
  rolling window on the client (traffic graphs). SSE `event:` names separate
  several kinds of event on one stream. Called in `<script setup>`, never
  in the template:
  `const traffic = useLiveSeries('/api/ont/42/traffic', { window: 300 })`,
  then `<LineChart :data="traffic" />`.
- `useApi()` is a thin fetch wrapper with typed results, loading and error
  state, and consistent error display through `ErrorState` and `Toast`.

### The Go-to-Vue contract (Weft owns it)

One record is described once, as a Go struct with `weft` tags; everything
else is generated from it.

```go
type Customer struct {
    ID     int64     `json:"id"     weft:"readonly"`
    Name   string    `json:"name"   weft:"required,max=120,label=Full name"`
    Email  string    `json:"email"  weft:"required,email"`
    Plan   string    `json:"plan"   weft:"enum=basic|pro|business"`
    Since  time.Time `json:"since"  weft:"label=Customer since"`
}
```

- **Wire format.** Lists: `{ "items": [...], "total": n, "page": p,
  "perPage": k }`; query parameters `page`, `perPage`, `sort` (`name`,
  `-since`), `q`, and `filter[field]=value`. Errors: `{ "error": { "code":
  "...", "message": "...", "fields": { "email": "must be an email" } } }`.
  Paging, sorting and filtering happen on the server.
- **`weft gen`** reads the tagged structs and writes `web/.weft/types.ts`
  (TypeScript types) and `web/.weft/schema.ts` (field schemas: label, kind,
  rules, options). `weft dev` reruns it when Go files change.
- **Validation in one place.** `weft.Validate(v)` in Go applies the same tags
  on the server; `Form` and `EntityDialog` apply the generated schema in the
  browser; server field errors land on the matching field. Weft defines the
  schema format itself, so no Zod or similar appears in app code.
- **`weft.Resource[T]`** in Go turns a storage implementation into list, get,
  create, update and delete handlers speaking the wire format:
  `api.Handle("/customers", weft.Resource[Customer](store))`. The store is a
  small interface (`List(ctx, Query)`, `Get`, `Create`, `Update`, `Delete`);
  Weft ships an in-memory one for the playground and tests, and apps bring
  their own database.
- **`weft add crud customers`** writes the struct, the store stub, the
  resource registration and the Vue page (a `DataTable` plus `EntityDialog`
  wired to the generated schema).
- `DataTable` takes a resource URL and does server-side paging, sorting and
  filtering through the wire format; it also accepts a plain array.

### Login and permissions (in the template)

- `weft.Auth` in Go: password login, sessions in signed HTTP-only cookies,
  logout, and `weft.Require("perm")` middleware. Users come from a small
  `UserStore` interface the app implements; the template ships an in-memory
  one with a seeded admin for development.
- Vue: a `useSession()` composable, a login page in the template, route meta
  `{ auth: true, perm: "customers.edit" }` enforced by the router guard, and a
  `perm` prop on `Button`, `Menu` items and `DataTable` row actions that hides
  or disables them.
- No database in the template.

## The CLI (`weft`, written in Go)

| Command | Does |
|---|---|
| `weft create <name>` | writes a new app from `templates/`, installs dependencies |
| `weft dev` | Vite and Go together, proxy, restart on change |
| `weft build` | Vite build, then `go build` with the frontend embedded |
| `weft add <block>` | adds a page or block scaffold (e.g. `weft add crud customers` writes Go and Vue) |
| `weft gen` | generates TypeScript types and field schemas from tagged Go structs |
| `weft upgrade` | moves the app to the installed Weft version |
| `weft catalog [name]` | prints the component catalogue, or one component |
| `weft check` | runs the Weft lint rules and type checks |
| `weft mcp` | runs an MCP server exposing the catalogue and `check` |

## Agent kit

- **Catalogue:** `catalog/components.json`, generated from component source and
  types: name, purpose, when to use and when not, props, events, slots, adaptive
  behaviour per axis, a minimal example. Single source for docs, `weft catalog`
  and the MCP server.
- **Skills** in `skills/`: building a Weft app, building a page from blocks,
  adding a component to Weft itself, theming, Go API plus live data.
- **Created apps** get an `AGENTS.md` and `CLAUDE.md` pointing at the catalogue
  and the rules.
- **Lint rules** (`@weft/eslint-plugin`, run by `weft check`): no raw
  interactive HTML elements where a Weft component exists (`<button>`,
  `<input>`, `<select>`, `<dialog>`), no third-party UI imports, no
  device-name checks, warn on long class lists in pages.

## Proof

The `playground/` app is the first product: a Go binary with a landing page and
an admin with a CRUD table, entity dialog, stat cards and a live-traffic chart
fed by SSE from mock data. It is the demo in the README and the CI smoke test,
checked at `pointer` and `touch` with both themes.

## Phases

1. Repository, README, licence, monorepo tooling, CI, install script and a
   first page at `weft.base.al`.
2. Tokens, themes, `useWeft()` adaptive core, layout tier.
3. Headless layer (focus, dismiss, roving focus, floating, typeahead), then
   primitives on it, each adaptive from the start, with keyboard and
   screen-reader tests, built as Vapor and as the virtual-DOM fallback.
4. Go `server` package, `@weft/vite`, `weft create`, `dev`, `build`; playground runs.
4b. Go-to-Vue contract: `weft gen`, `weft.Validate`, `weft.Resource`, `Form`
   on generated schemas; `weft.Auth` and the template's login.
5. App blocks (`DataTable` on the wire format, `EntityDialog`, `weft add crud`)
   and charts with `useLive`.
6. Agent kit: catalogue, skills, lint rules, `check`, `mcp`.
7. Landing blocks; first public release (`0.1.0`).

Alpha releases (`0.0.x`) are cut from the end of phase 4. Before that,
`weft create --local` points the app at the working repository.

### Distribution: one source, no npm publishing

Weft is never published to npm. Everything ships as one versioned thing:

- `curl -fsSL https://weft.base.al/install | sh` downloads the `weft` binary
  for the platform from the GitHub release and puts it on the path. (The
  domain resolves today but serves nothing yet; the install script and site
  are part of phase 1.)
- The binary embeds Weft's JavaScript packages (`ui`, `vite`, `themes`,
  `eslint-plugin`) and the templates, built at release time with `go:embed`.
- `weft create` and `weft upgrade` unpack them into `~/.weft/<version>/`, and
  the app's `package.json` refers to them with `file:` paths, so the CLI
  version decides the Weft version. `weft.config.ts` records the version;
  `weft upgrade` moves an app to the installed one.
- The Go module stays `github.com/flakerimi/weft`, fetched from GitHub by Go
  as usual, tagged with the same version.
- Third-party packages (Vue, Vite, Floating UI, Unovis) still come from the
  public npm registry through the package manager. No account or org is
  needed for that.
- `weft.base.al` also hosts the docs and the component catalogue.

Cost: Weft cannot be added to an existing Vite app with a package install;
`weft add-to` (later) would copy it in. Accepted, since Weft targets new apps.

### Public pages and search engines

No per-request rendering. For landing routes, Go fills the title,
description and link-preview meta tags into `index.html` per route, from
route metadata emitted by the build, so link previews and basic indexing work
without Node. Full build-time prerendering of landing routes is a later
option, not in the first release.

## Decided by the owner

- Own components and API, no styled kit underneath.
- Vapor-first, with Weft's own headless layer instead of Reka.
- Vue, not Svelte or Solid. Go backend embeds the Vue app instead of Nuxt.
- Neutral core with Lisa as one theme.
- Name Weft; repository `flakerimi/weft`; plain component names without
  prefix.
- No npm publishing: one source, installed with curl from `weft.base.al`,
  hosted on the Lisa server.
- New apps only; existing apps are out of scope.
- README, skills, tools and CLI are part of the project.

## Calls made by Claude (open to correction)

- Licence MIT.
- pnpm workspace inside the repository; CLI in Go.
- Tailwind used inside components but compiled away for consumers.
- Width axis taken from the container, not the window.
- Charts as an optional subpath so apps without charts do not pull in Unovis.
- MCP server as a CLI subcommand rather than a separate package.
- Playground doubles as the proof app and the CI smoke test.
- `useLive` replaces state, `useLiveSeries` appends with a client-side window.
- `lisa` theme generated from `Lisa/docs/contract/tokens.json`, and published
  with the library (the code is MIT; the Lisa name and brand are not granted
  by the licence, noted in the theme's README).
- Alphas from phase 4, lockstep versions, curl installer plus `go install`.
- Weft owns the Go-to-Vue contract (wire format, `weft gen`, `weft.Resource`,
  shared validation), with server-side paging.
- Templates include login, sessions, permissions; no database.
- GitHub repository private until `0.1.0`, then public.
- Per-route meta tags injected by Go for landing pages; no prerendering yet.

## Left to the builder

Internal structure of each component, test tooling (Vitest plus a browser
runner), the exact icon set, docs-site tooling, the SSE event format, the
release tooling.
