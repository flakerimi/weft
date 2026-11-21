---
title: CLI
description: Every command and flag of weft and weft-build-pages, and the installer's settings.
order: 30
---

Look up here every command and flag the `weft` command has today, the page build's flags, and the installer's settings. Commands the design plans but the code does not have yet are under [Coming next](../coming-next.md).

## weft

```text
weft — one UI system for Go and Vue applications

Usage: weft <command>

Commands:
  update     update weft to the latest release (--check, --version X)
  version    print the Weft version
```

`weft`, `weft help`, `weft -h` and `weft --help` print this. An unknown command prints `weft: unknown command "x". Run "weft help".` and exits with status 2.

### weft version

Prints the version and the platform it was built for:

```text
weft 0.0.3 (darwin/arm64)
```

A build that is not a release says `0.0.0-dev`.

### weft update

Replaces the running `weft` with another release from `weft.base.al`.

| Flag | What it does |
| --- | --- |
| `--check` | Only says whether the latest release differs from yours, and changes nothing. After a rollback of the site the latest may be older than yours. |
| `--version X` | Installs release `X` instead of the latest, older ones included |

It downloads the archive for your platform and `SHA256SUMS`, checks the archive against it, writes the new binary beside the running one, and renames it into place, so the old one stays whole until the new one is complete. It follows a symlink to the real file.

It says `weft X is up to date.` when there is nothing to do, and `Updated weft X → Y.` when it is done. Any failure is one sentence on standard error, with exit status 1; a flag it does not know exits with status 2. A development build refuses to update and tells you to run the installer.

| Variable | What it does |
| --- | --- |
| `WEFT_BASE` | Where the releases are. Default `https://weft.base.al` |

## The installer

`curl -fsSL https://weft.base.al/install | sh` installs the latest release. It runs on macOS and Linux, on arm64 and amd64, and needs `curl`, and `sha256sum` or `shasum`.

| Variable | What it does | Default |
| --- | --- | --- |
| `WEFT_VERSION` | The release to install | the latest |
| `WEFT_BIN_DIR` | Where to put `weft` | `$HOME/.local/bin` |
| `WEFT_BASE` | Where the releases are | `https://weft.base.al` |

## weft-build-pages

The page build. It comes with `@weft/vite`; run it from a site's `package.json` script.

```text
weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa|weft] [--preload-font <file.woff2>]...
```

| Flag | What it does |
| --- | --- |
| `--root` | The site's folder, which holds `pages/`. Required. |
| `--out` | Where the build goes. Required. It must be missing, empty, or a folder an earlier build wrote and nobody changed since; anything else is refused. |
| `--theme` | `neutral` (the default), `lisa` or `weft`. See [Themes](../guides/themes.md). |
| `--app-prefix` | A path prefix, such as `/app`, for a client-rendered app built beside the pages. |
| `--app-dir` | Where that app's Vite build is. Default `<root>/app-dist`. |
| `--preload-font` | A font file, such as `lucide.woff2`, to preload on the pages whose CSS names it, besides the Latin faces of the families they use. Give it once per font. |

It prints `weft: built N pages into <dist>`. It needs Node 22.18 or later, which runs TypeScript without a build step. Any error is one sentence, and the folder at `--out` is left as it was.
