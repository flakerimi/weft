---
title: Coming next
description: What the design plans and the code does not have yet, by phase.
order: 40
---

Check here whether something you want is built yet: this is what Weft's design plans and the code does not have, so you know what not to look for. Each item names its phase: a number is a phase of the Weft design (`docs/superpowers/specs/2026-11-20-weft-design.md`), and W2 to W7 are phases of the Warp design (`docs/superpowers/specs/2026-11-23-warp-design.md`), which replaces the Weft design's phases 4b and 5. A first pass through W2 to W4, one slice end to end, is built (`docs/superpowers/specs/2026-11-25-warp-slice-design.md`); what it left out is listed under each. Nothing listed here works today.

## Commands

| Command | What it will do | Phase |
| --- | --- | --- |
| `weft db` | Show how the database differs from the app's records | W3 |
| `weft check` | Run Weft's lint rules and type checks | 6 |
| `weft mcp` | Serve the catalogue and `check` to agents over MCP | 6 |
| `weft deploy` | Publish an app | 6b |

The commands that make, run, build and upgrade an app, `weft scaffold`, `weft gen` and `weft routes`, and `weft catalog`, which prints the component catalogue, work today: see the [CLI reference](reference/cli.md).

## App routes

A client-side router and `layouts/`, for an app whose pages the browser switches between without loading a new document, inside a layout they share. Phase 4c. Today an app made with `weft create` is pages rendered at build, with islands, and a Go API; a page for a record made after the build is a [run-time page](guides/pages-and-islands.md#pages-for-records-made-after-the-build), and a layout is a component the pages share.

## Types from Go

`weft gen` and `weft routes` work today ([Types from Go](guides/types-from-go.md)). Not yet, phase W2 widened:

- the generated client calling an API on another origin than the pages: `setBase` takes a path prefix or a URL of the pages' own origin.

## Data

`warp/data` works today, with SQLite ([Data](guides/data.md)). Not yet, phase W3:

- PostgreSQL, then MySQL;
- relations (`belongs_to`, `has_many`, `many_to_many`) and soft delete;
- `weft db`.

## Login and permissions

The `auth` module works today, with one role, the owner ([Login](guides/login.md)). Not yet, phase W4:

- registration, invitations and password reset, with the `mail` module, which sends email through SMTP or prints it to the log;
- roles other than the owner, and permissions given to them;
- a session table on the server, for signing out everywhere and ending one session;
- `useSession()` in `@weft/ui`, `perm` on `Button` and row actions, and a route guard in Vue. An app `weft create` makes has its own `web/session.ts`, with its login and setup pages.

## Scaffolding and app blocks

`weft scaffold`, the app template, `DataTable` (server-side paging, sorting, debounced search and filtering over the list shape) and `EntityDialog` (creating and editing records from a form schema) work today ([CLI](reference/cli.md#weft-scaffold), [Components](/components)). Not yet, Weft 0.2.0:

- date and time fields in `EntityDialog` and in the pages `weft scaffold` writes: a date control, and the schema saying a field is a time. Until then `weft scaffold` takes `date` and `datetime` only with `--api`.

## The browser's rules and Go's

`Form` and Go's validation differ on a few values, such as a `0` with `required` or `min` (see [Types from Go](guides/types-from-go.md#where-the-browser-and-go-differ)). The server's answer lands on the field. The designs give this no phase.

## Attachments and live data

Phase W6:

- the `storage` module: attachments on a record's `image` and `file` fields, served under `/api/files/`, with `FileUpload` bound to the field;
- the `events` module (`events.On` and `Emit`), and the `live` module: server-sent events at `/api/live`, read in Vue with `useLive` and `useLiveSeries`;
- charts (`LineChart`, `AreaChart`, `BarChart`, `DonutChart`, `Sparkline`) that take live data. The Weft design put them in phase 5 with `useLive`; the Warp design, which replaces that phase, gives them none, so they are listed here, inferred, with `useLive`.

## More modules

Phase W7: `jobs`, for work on a schedule; `i18n`, for translated fields; `middleware`, for request logging, security headers, CORS, a rate limit and API keys; and `docs`, an OpenAPI document at `/api/openapi.json`.

## Errors

- When a JSON body has both a field of the wrong type and, inside a nested object, a field the route does not take, the unknown field is reported only when it comes before the wrong-typed value; otherwise the answer names the wrong type.
- A wrong-typed value is named without a list's index, and a map's by the map's own key: `{"lines":[{"qty":1},{"qty":"x"}]}` gives `lines.qty: Send a whole number.`, where validation would say `lines.1.qty`.
- A body with two problems names one field, which may differ by build: `{"n8":300,"when":5}` names `when` on the default build and `n8` under `GOEXPERIMENT=jsonv2`.
- A taken port is named as taken (`warp: port 8080 is taken; …`) on macOS, Linux and Windows (`WSAEADDRINUSE`). Windows is not yet a supported platform: the release workflow builds `weft` for Windows and runs the Go tests there (`weft dev` stops its process trees with a Job Object), while the Windows installer and published Windows release archives have not landed yet.

The Warp design gives these no phase.

## The agent kit

Lint rules (`@weft/eslint-plugin`) that flag a raw `<button>` where a Weft component exists, `weft check` to run them, and `weft mcp`. Phase 6.

## Site features

An `<Image>` component, actions, environment values, a development overlay and `weft deploy`. Phase 6b. Content collections, the first of these, already work: see [Markdown content](guides/markdown.md).

## More landing blocks

`Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq` and `Cta` work today ([Components](/components)). Not yet, phase 7: `Carousel`, `Marquee`, `FooterColumns`, `PageCard`, `PageColumns`, `PageList`, `PageLogos` and `PricingTable`.
