---
title: Weft docs
description: What Weft is today, what it is made of, and where to start.
order: 1
---

These docs show you how to build an app with Weft, serve it from one Go binary, and publish it. They describe what the code does today. What is not built yet is listed under [Coming next](coming-next.md).

## What Weft is

Weft is one UI system for Go and Vue apps. It has four parts:

- **`@weft/ui`**: 82 Vue components, written once, that adapt to touch or pointer, to a landing page or an app, and to the width they are given. Every component is a Vapor component: no virtual DOM.
- **`@weft/vite`**: the page build. It renders your pages to HTML when you build, compresses them, and ships JavaScript only for the parts you mark interactive.
- **Warp, the Go package** `github.com/flakerimi/weft/warp`: an app made of modules, with typed routes that check their input and answer in one error shape, records in SQLite, a login, and `warp.Serve`, which serves the page build from memory next to that API, in one binary.
- **The `weft` command**: it makes an app, runs it while you work, writes the TypeScript types and client from its Go, builds it into one binary, and updates itself. It carries the other three parts.

Weft is pre-alpha. Its packages are not published to npm: the `weft` command puts them in each app it makes. The repository is private for now.

## Where to start

1. [Install](getting-started/install.md) the command.
2. [Create an app](getting-started/create-an-app.md), run it while you edit it, and build it into one binary.
3. [Add a page](getting-started/first-site.md) to it.
4. [Add a Go route](getting-started/first-go-handler.md) that checks what it is sent.

Then read the guides as you need them:

- [Pages and islands](guides/pages-and-islands.md): routes, the head, and the interactive parts.
- [Markdown content](guides/markdown.md): docs like these, checked at build time.
- [Warp, the Go side](guides/warp.md): the app, its modules, `.env`, a proxy, cross-site requests and tests.
- [Typed routes](guides/routes.md): routes, their input, validation and errors.
- [Data](guides/data.md): records in SQLite, and the five routes of a resource.
- [Login](guides/login.md): the owner, sessions and permissions.
- [Types from Go](guides/types-from-go.md): the typed client and form rules `weft gen` writes.
- [Serving from Go](guides/go-serving.md): what `warp.Serve` sends, and when.
- [Themes](guides/themes.md): colours, type and spacing as tokens.
- [The adaptive model](guides/adaptive.md): input, surface and width.
- [Publishing](guides/publishing.md): one binary on a server.

The [CLI](reference/cli.md) and [Go package](reference/go.md) references list every command, flag and function.

## An example app

The repository's `examples/shop` is a small back office built the way these docs say: a login, a products table with search, a filter and a dialog to make and change one, and a page per product, with no fetch or HTTP handler written by hand. `scripts/shop.sh` runs it with the repository's Weft (`scripts/shop.sh build` builds it, `scripts/shop.sh test` runs its tests), and the full `go test ./cmd/weft` builds a copy, runs its tests and type check, and requests its pages and API (`-short` skips it).

## This site

This site is built with Weft. Its pages, these docs included, are rendered when it is built and served by a Go binary that uses `warp.Serve`. A docs page loads no JavaScript, except a copy button beside a command to copy. Its source is `site-src/` in the repository.

For agents, the site has [`/llms.txt`](/llms.txt), an index of every doc, component and skill, and Claude Code skills such as [weft-site](/skills/weft-site), which teach an agent to build with Weft. [The skills page](/skills) shows how to install them.
