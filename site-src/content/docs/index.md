---
title: Weft docs
description: What Weft is today, what it is made of, and where to start.
order: 1
---

These docs show you how to build a site with Weft, serve it from one Go binary, and publish it. They describe what the code does today. What is not built yet is listed under [Coming next](coming-next.md).

## What Weft is

Weft is one UI system for Go and Vue apps. It has four parts:

- **`@weft/ui`**: 76 Vue components, written once, that adapt to touch or pointer, to a landing page or an app, and to the width they are given. Every component is a Vapor component: no virtual DOM.
- **`@weft/vite`**: the page build. It renders your pages to HTML when you build, compresses them, and ships JavaScript only for the parts you mark interactive.
- **The `weft` Go package**: `weft.Serve` serves that build from memory, next to your API, in one binary.
- **The `weft` command**: today it prints its version and updates itself.

Weft is pre-alpha. The packages are not published to npm, and the repository is private for now, so you build inside a clone of it. The [install page](getting-started/install.md) says how.

## Where to start

1. [Install](getting-started/install.md) the command and the repository.
2. [Build your first site](getting-started/first-site.md): one page, rendered at build time.
3. [Add a Go handler](getting-started/first-go-handler.md) and serve both from one binary.

Then read the guides as you need them:

- [Pages and islands](guides/pages-and-islands.md): routes, the head, and the interactive parts.
- [Markdown content](guides/markdown.md): docs like these, checked at build time.
- [Serving from Go](guides/go-serving.md): what `weft.Serve` sends, and when.
- [Themes](guides/themes.md): colours, type and spacing as tokens.
- [The adaptive model](guides/adaptive.md): input, surface and width.
- [Publishing](guides/publishing.md): one binary on a server.

The [CLI](reference/cli.md) and [Go package](reference/go.md) references list every command, flag and function.

## This site

This site is built with Weft. Its pages, these docs included, are rendered when it is built and served by a Go binary that uses `weft.Serve`. A docs page loads no JavaScript, except a copy button beside a command to copy. Its source is `site-src/` in the repository.

For agents, the site has [`/llms.txt`](/llms.txt), an index of every doc, component and skill, and Claude Code skills such as [weft-site](/skills/weft-site), which teach an agent to build with Weft. The header's For agents link shows how to install them.
