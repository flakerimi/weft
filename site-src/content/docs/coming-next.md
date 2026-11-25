---
title: Coming next
description: What the design plans and the code does not have yet, by phase.
order: 40
---

Check here whether something you want is built yet: this is what Weft's design plans and the code does not have, so you know what not to look for. Each item names its phase: a number is a phase of the Weft design (`docs/superpowers/specs/2026-11-20-weft-design.md`), and W2 to W7 are phases of the Warp design (`docs/superpowers/specs/2026-11-23-warp-design.md`), which replaces the Weft design's phases 4b and 5. Nothing here works today.

## Commands

| Command | What it will do | Phase |
| --- | --- | --- |
| `weft gen` | Write the TypeScript types, a typed function per API route and the form schemas, from the app's Go types | W2 |
| `weft routes` | Print every route, with the permission it needs | W2 |
| `weft db` | Show how the database differs from the app's records | W3 |
| `weft scaffold` | Write a resource from a name and typed fields: its Go module, its tests and a page to list and edit it | W5 |
| `weft check` | Run Weft's lint rules and type checks | 6 |
| `weft mcp` | Serve the catalogue and `check` to agents over MCP | 6 |
| `weft deploy` | Publish an app | 6b |

The commands that make, run, build and upgrade an app, and `weft catalog`, which prints the component catalogue, work today: see the [CLI reference](reference/cli.md).

## App routes

A client-side router and `layouts/`, for an app whose pages the browser switches between without loading a new document, inside a layout they share. Phase 4c. Today an app made with `weft create` is pages rendered at build, with islands, and a Go API.

## Types from Go

The app describes its own routes and types, and the Vue side gets them. Phase W2:

- `warp/describe`, which lists every route with its method, path, input type and output type;
- `weft gen`, which writes `web/.weft-types/api.ts`: the TypeScript types, a typed function per route, and a form schema per input, with its rules, which `Form` checks in the browser;
- `weft routes`, and `weft dev` running `weft gen` again on each Go change.

Today the routes, their inputs and their rules are in Go ([Warp, the Go side](guides/warp.md)), and the Vue side's types are written by hand.

## Data

`warp/data`: records stored through `database/sql`, with no ORM. Phase W3:

- `data.Store[T]`, with `List`, `Get`, `Create`, `Update` and `Delete` for a struct, its columns read from its tags;
- tables and columns added at start, never dropped, and `weft db`;
- lists in one shape, `{"items": [...], "total": 42, "page": 1, "per_page": 20}`, with `page`, `per_page`, `sort`, `q` and `filter[field]` in the query;
- `warp.Resource`, which registers the list, show, create, update and delete routes on a store;
- updates as `PATCH` with fields that are present or absent, so an update can clear a field;
- the `unique` rule checked against the database: today it is accepted and nothing checks it;
- SQLite, then PostgreSQL, then MySQL.

## Login and permissions

The `auth` module, switched on in `.env`. Phase W4:

- users, sessions in a signed HttpOnly cookie, roles and permissions, and `auth.Can("products:edit")` on a route;
- `useSession()`, a login page, `perm` on `Button` and row actions, and a route guard in Vue;
- the `mail` module, which sends email through SMTP or prints it to the log.

## Scaffolding and app blocks

Phase W5, with the blocks the Weft design had in its phase 5:

- `weft scaffold`, which writes `app/<name>/` (the model, the routes and their tests), adds it to `app/modules.go`, and writes a page for it;
- `DataTable`, with server-side paging, sorting and filtering over the list shape, and `EntityDialog`, which creates and edits records from a form schema.

## Attachments and live data

Phase W6:

- the `storage` module: attachments on a record's `image` and `file` fields, served under `/api/files/`, with `FileUpload` bound to the field;
- the `events` module (`events.On` and `Emit`), and the `live` module: server-sent events at `/api/live`, read in Vue with `useLive` and `useLiveSeries`;
- charts (`LineChart`, `AreaChart`, `BarChart`, `DonutChart`, `Sparkline`) that take live data. The Weft design put them in phase 5 with `useLive`; the Warp design, which replaces that phase, gives them none, so they are listed here, inferred, with `useLive`.

## More modules

Phase W7: `jobs`, for work on a schedule; `i18n`, for translated fields; `middleware`, for request logging, security headers, CORS, a rate limit and API keys; and `docs`, an OpenAPI document at `/api/openapi.json`.

## Errors

- When a JSON body has both a field of the wrong type and, inside a nested object, a field the route does not take, the unknown field is reported only when it comes before the wrong-typed value; otherwise the answer names the wrong type.
- A wrong-typed value inside a list or map is reported without its index or key, as `lines.qty`, where validation says `lines.1.sku`. The form schemas of phase W2 need the index.
- A taken port is named as taken (`warp: port 8080 is taken; …`) on macOS and Linux only; elsewhere it is the general `cannot listen` sentence.

The Warp design gives the first and the last no phase.

## The agent kit

Lint rules (`@weft/eslint-plugin`) that flag a raw `<button>` where a Weft component exists, `weft check` to run them, and `weft mcp`. Phase 6.

## Site features

An `<Image>` component, actions, environment values, a development overlay and `weft deploy`. Phase 6b. Content collections, the first of these, already work: see [Markdown content](guides/markdown.md).

## Landing blocks

`Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq` and `Cta`. Phase 7, with the first public release, `0.1.0`.
