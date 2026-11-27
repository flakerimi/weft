---
title: Page tools
description: Every flag and error of weft-build-pages, weft-dev and weft-types, the commands of @weft/vite that weft runs.
order: 37
---

`weft build`, `weft dev` and `weft create` run these for an app ([CLI](cli.md)); a site without the Go side runs them from its `package.json` scripts.

## weft-build-pages

The page build. It comes with `@weft/vite`; run it from a site's `package.json` script.

```text
weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa|weft] [--preload-font <file.woff2>]... [--separate-runtime <section path>]...
```

| Flag | What it does |
| --- | --- |
| `--root` | The site's folder, which holds `pages/`. Required. |
| `--out` | Where the build goes. Required. It must be missing, empty, or a folder an earlier build wrote and nobody changed since; anything else is refused. |
| `--theme` | `neutral` (the default), `lisa` or `weft`. See [Themes](../guides/themes.md). |
| `--app-prefix` | A path prefix, such as `/app`, for a client-rendered app built beside the pages. |
| `--app-dir` | Where that app's Vite build is. Default `<root>/app-dist`. |
| `--preload-font` | A font file, such as `lucide.woff2`, to preload on the pages whose CSS names it, besides the Latin faces of the families they use. Give it once per font. |
| `--separate-runtime` | A section of the site, such as `/components`, whose pages' islands get a runtime of their own, with only the Vue they use, so heavy islands do not grow the shared one. A page is in the longest section it is in or under: `/components/button` is in `/components`. Give it once per section, as a path with no trailing `/`. |

It prints `weft: built N pages into <dist>`. It needs Node 22.18 or later, which runs TypeScript without a build step. Any error is one sentence; the folder at `--out` is left as it was.

## weft-dev

The page dev server. It comes with `@weft/vite`.

```text
weft-dev --root <site> [--port n] [--host h] [--api <url>] [--theme neutral|lisa|weft] [--exit-with-parent]
```

| Flag | What it does |
| --- | --- |
| `--root` | The site's folder, which holds `pages/`. Required. Its `package.json` may be in a folder above. |
| `--port` | The port to listen on. Default `5173`; `0` takes a free one. |
| `--host` | The host to listen on. Default `localhost`. Any other exposes Vite's file serving, the app's folder included, to the network. |
| `--api` | Where `/api` and every path under it go, such as `http://127.0.0.1:8081`. Without it they are pages like any other. |
| `--theme` | `neutral` (the default), `lisa` or `weft`, as for the build. |
| `--exit-with-parent` | Stop, with exit status 0, when standard input ends. `weft dev` passes it with a pipe it holds open. Without it standard input is not read. |

It prints `weft: dev server at <url>` once it listens. Each page is rendered when asked for, as the build renders it, so a saved edit shows on the next request; a dynamic page's `getPaths()` is called then too. A page that exports `runtime = true` is served, as its shell, for every path its pattern matches that no other page gives. An unknown path gets `pages/404.vue` with status 404, or a plain 404 without one. A page that fails to render gets a 500 page naming its file and the error, and reloads when you save the fix; the server keeps running. A change to any file under `--root` reloads the browser. A file a page writes with `useFile` is served once that page has rendered.

It stops on SIGTERM or Ctrl-C. A port in use is the sentence `weft dev: port <n> is in use; stop what is using it or pass --port.`, with exit status 1; a flag it does not know exits with status 2.

## weft-types

Writes a site's `.weft-types/components.d.ts`, which tells TypeScript what each auto-imported component is, as a build does, without building. It comes with `@weft/vite`; `weft create`, `weft build`, `weft dev` and `weft upgrade` run it in an app.

```text
weft-types --root <site>
```

It prints nothing once the file is written. When `@weft/ui` cannot be found from the site, it says so with exit status 1.
