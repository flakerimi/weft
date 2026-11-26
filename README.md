<h1 align="center">weft</h1>

<p align="center">
  <strong>One UI system for Go and Vue apps.</strong><br>
  Vapor-first components that know whether they're under a finger or a mouse,<br>
  on a landing page or in an app — shipped inside a single Go binary.
</p>

<p align="center">
  <code>pre-alpha</code> · <code>Vue 3.6 Vapor</code> · <code>Go 1.26</code> · <code>MIT</code>
</p>

---

```sh
curl -fsSL https://weft.base.al/install | sh
weft create myapp      # an app: Vue pages and a Go API
cd myapp && weft dev   # run it while you work
weft build             # one binary that serves both
weft update            # later: move to the newest release
```

## Why

Every product rebuilds the same pieces: a stat card, a table with actions and
paging, an edit dialog, a confirm prompt, a sidebar. Each app drifts a little,
and every agent that builds a screen hand-assembles it from divs again.

Weft is the set of pieces, built once:

- **Yours, all the way down.** Every component, API and style is Weft's. No
  styled kit underneath, no third-party names in your app code.
- **Adaptive by default.** You write `<DatePicker v-model="day" />` once. On a
  mouse it is a popover calendar; under a finger it is a sheet with big targets.
  No `MobileDatePicker` to remember. (The primitives land next; the layout
  components adapt today.)
- **Go-first.** One binary serves your API and embeds your Vue app. No Node in
  production.
- **Built for agents.** Big building blocks, one way to do each thing, and a
  machine-readable catalogue so an agent composes screens instead of guessing.
- **Vapor-first.** Components compile with Vue's Vapor mode: no virtual DOM.

## A taste

```go
// app/stats/routes.go: a module with one typed route, GET /api/stats/{id}.
package stats

import (
	"context"
	"fmt"

	"github.com/flakerimi/weft/warp"
)

type Stat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type StatIn struct {
	ID int `path:"id"`
}

type module struct{}

func (module) Name() string { return "stats" }

func (module) Register(app *warp.App) error {
	warp.Get(app, "/stats/{id}", show)
	return nil
}

func Module() warp.Module { return module{} }

func show(ctx context.Context, in StatIn) (Stat, error) {
	if in.ID != 1 {
		return Stat{}, warp.NotFound(fmt.Sprintf("No stat has id %d.", in.ID))
	}
	return Stat{Name: "ONTs online", Value: 1284}, nil
}
```

```vue
<script setup vapor lang="ts">
import { Card, Grid, Page } from '@weft/ui'
const stats = [{ name: 'ONTs online', value: 1284 }, { name: 'Open tickets', value: 7 }]
</script>

<template>
  <Page title="Network" description="Live from every ONT.">
    <Grid>
      <Card v-for="s in stats" :key="s.name">
        <template #header>{{ s.name }}</template>
        {{ s.value }}
      </Card>
    </Grid>
  </Page>
</template>
```

The Go side is Warp (`github.com/flakerimi/weft/warp`): an app is a list of
modules, each route is a typed Go function whose input is filled from the
request and validated before it runs, and every error answers in one JSON
shape. `warp.Serve` serves the built pages from memory, already compressed,
beside the API under `/api`.

## How it adapts

Two independent axes, read by every component through `useWeft()`:

| | **pointer** (mouse, trackpad) | **touch** (finger) |
|---|---|---|
| **app** | dense: 36 px controls, 14 px text, popovers | 44 px targets, sheets instead of popovers |
| **landing** | roomy: 18 px text, capped page width | roomy and touch-sized |

Input comes from what the browser reports (`(pointer: coarse)`), never from a
device name, and follows changes live. Surface is declared by the layout:
`<WeftProvider surface="landing">`. Width comes from each component's own
container, so a grid in a side panel lays out narrow on a big screen.

<p align="center">
  <img src="docs/assets/playground-neutral.png" width="49%" alt="Weft playground, neutral theme, light">
  <img src="docs/assets/playground-lisa.png" width="49%" alt="Weft playground, lisa theme, dark">
</p>

## Themes

A theme is one token file: colours for light and dark, radius, spacing, type,
motion. Weft ships `neutral`, `lisa` (generated from LisaOS's token
contract) and `weft` (weft.base.al's). Components read only `--weft-*` custom properties, so a brand is a
stylesheet, not a fork. Want Tailwind for your own components? Import
`@weft/themes/tailwind.css` and its colours are Weft's.

## Status

Pre-alpha. What exists today:

- `weft` CLI (`create`, `dev`, `build`, `gen`, `routes`, `unpack`,
  `upgrade`, `catalog`, `version`, `update`), which carries Weft's packages, and the curl installer
- tokens and the `neutral`, `lisa` and `weft` themes
- the adaptive core: `useWeft`, `WeftProvider`, `useWidth`
- the Vapor components of `@weft/ui`, from layout and forms to dates,
  overlays, navigation and a table, on a headless layer for focus, dismissal,
  roving focus, floating and typeahead
- `@weft/vite`: the page build, server rendering, islands, Markdown content
  and a dev server
- Warp, the Go side (`github.com/flakerimi/weft/warp`): an app of modules,
  typed routes with validation, one error shape, `.env` configuration, and
  `warp.Serve` for the embedded pages; records in SQLite (`warp/data`), a
  login with one owner (`warp/auth`) and a test kit (`warp/warptest`); and a
  playground that uses all of it
- `weft gen`, which writes the TypeScript types, a typed client and the form
  rules from the app's Go types, and `weft routes`
- `examples/shop`, a back office with a login, a products table and a page
  per product, which `scripts/shop.sh` runs and the suite drives
- weft.base.al: the docs, the component gallery, the skills, and the files
  for agents, `/index.md` (the page to point an agent at, from the next
  release) and `/llms.txt`

Next: widening the Warp slice (more database engines and relations;
registration, reset, roles and mail; `weft scaffold` with app blocks; the app
template taking the shop's shape), then Warp's W6 and W7 (attachments, live
data and more modules), app routes, the rest of the
agent kit (lint rules, `weft check`, MCP), site features, and landing blocks.
The designs are
[`docs/superpowers/specs/2026-11-20-weft-design.md`](docs/superpowers/specs/2026-11-20-weft-design.md)
and, for the Go side,
[`docs/superpowers/specs/2026-11-23-warp-design.md`](docs/superpowers/specs/2026-11-23-warp-design.md)
and
[`docs/superpowers/specs/2026-11-25-warp-slice-design.md`](docs/superpowers/specs/2026-11-25-warp-slice-design.md).

## Develop

Needs Go 1.26, Node 22.18+ and pnpm 10.

```sh
pnpm install
scripts/playground.sh        # build the web app, run the Go binary on :8080
scripts/site.sh              # build weft.base.al from site-src, run it on :8080
pnpm test && go test ./...   # the Go tests need all three builds first (below)
```

`go test ./...` compiles the three Go binaries that embed a build, so build all
three first: `scripts/playground.sh build`, `scripts/site.sh build` and
`pnpm --filter weft-bench build`. `pnpm test` builds `@weft/themes` and
`@weft/ui` itself, since the page build tests read their CSS.

## License

MIT. The `lisa` theme's values come from LisaOS; the Lisa name and brand are
not granted by this licence.
