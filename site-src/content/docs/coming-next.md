---
title: Coming next
description: What the design plans and the code does not have yet, by phase.
order: 40
---

Check here whether something you want is built yet: this is what Weft's design plans and the code does not have, so you know what not to look for. Each item names its phase in the design (`docs/superpowers/specs/2026-11-20-weft-design.md`, Phases). Nothing here works today.

## Commands

| Command | What it will do | Phase |
| --- | --- | --- |
| `weft create <name>` | Write a new app from a template and install its dependencies | 4 |
| `weft gen` | Write TypeScript types and field schemas from tagged Go structs | 4b |
| `weft add <block>` | Add a page or block, such as `weft add crud customers` | 5 |
| `weft catalog [name]` | Print the component catalogue, or one component | 6 |
| `weft check` | Run Weft's lint rules and type checks | 6 |
| `weft mcp` | Serve the catalogue and `check` to agents over MCP | 6 |
| `weft deploy` | Publish an app | 6b |

Until then, build sites inside the repository as [Your first site](getting-started/first-site.md) shows, with `weft-build-pages` and `go build`. The catalogue itself exists today, as `catalog/components.json` in the repository.

## Packages in the binary

The `weft` binary will carry Weft's JavaScript packages, and `weft create` will unpack them, so apps need neither npm nor a clone of the repository. Phase 4, inferred: the design gives this no phase of its own, and `weft create`, which unpacks them, is phase 4.

## Tailwind in a new app

`weft create` will set up `@weft/themes/tailwind.css` in an app that wants Tailwind for its own components. Phase 4, with `weft create`. The file itself works today: see [Themes](guides/themes.md#tailwind).

## The Go-to-Vue contract

Describe a record once, as a Go struct with `weft` tags, and get the rest from it. Phase 4b:

- `weft gen` writes the TypeScript types and field schemas;
- `weft.Validate` checks a value against the same tags on the server, and `Form` checks the generated schema in the browser;
- `weft.Resource[T]` turns a store into list, get, create, update and delete handlers that speak one wire format for lists, paging, sorting, filtering and errors.

## Login and permissions

`weft.Auth` for password login and sessions in signed cookies, `weft.Require("perm")` middleware, a `useSession()` composable, and a login page in the template. Phase 4b.

## App blocks and live data

Phase 5:

- `DataTable`, with server-side paging, sorting and filtering over the wire format, and `EntityDialog`, which creates and edits records from a field schema;
- charts (`LineChart`, `AreaChart`, `BarChart`, `DonutChart`, `Sparkline`) that take live data;
- `useLive` and `useLiveSeries` in Vue, for server-sent events, and `weft.SSE` in Go to send them. The design puts `useLive` in phase 5 and gives `weft.SSE` no phase; it is listed here, inferred, because `useLive` reads it.

## The agent kit

Lint rules (`@weft/eslint-plugin`) that flag a raw `<button>` where a Weft component exists, `weft check` to run them, and `weft mcp`. Phase 6.

## Site features

An `<Image>` component, actions, environment values, a development overlay and `weft deploy`. Phase 6b. Content collections, the first of these, already work: see [Markdown content](guides/markdown.md).

## Landing blocks

`Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq` and `Cta`. Phase 7, with the first public release, `0.1.0`.
