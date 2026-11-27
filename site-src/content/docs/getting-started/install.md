---
title: Install
description: Install the weft command, and get the packages you build with.
order: 10
---

On this page you install the `weft` command and the tools an app needs. You need macOS or Linux, on arm64 or amd64.

## The weft command

Run the installer:

```sh copy
curl -fsSL https://weft.base.al/install | sh
```

It downloads the release for your system from `weft.base.al`, checks it against the release's `SHA256SUMS`, and puts `weft` in `~/.local/bin`. If that folder is not on your `PATH`, it tells you. Each release is also a [GitHub Release](https://github.com/flakerimi/weft/releases), `v<version>`, with the same archives and the same `SHA256SUMS`.

Check that it works:

```sh
weft version
```

It prints the version and your platform, such as `weft 0.0.3 (darwin/arm64)`.

### Installer settings

Set these before `sh` to change what the installer does:

| Variable | What it does | Default |
| --- | --- | --- |
| `WEFT_BIN_DIR` | Where to put `weft` | `$HOME/.local/bin` |
| `WEFT_VERSION` | Which release to install | the latest |
| `WEFT_BASE` | Where the releases are | `https://weft.base.al` |

For example, to install release 0.0.2 into `~/bin`:

```sh
curl -fsSL https://weft.base.al/install | WEFT_VERSION=0.0.2 WEFT_BIN_DIR="$HOME/bin" sh
```

### Update

`weft update` replaces the binary with the latest release. `weft update --check` only tells you whether there is a newer one. Then run `weft upgrade` in each app to move it to that release: its packages, and the version of Warp its `go.mod` requires. See the [CLI reference](../reference/cli.md) for the flags.

## What an app needs

The `weft` command carries Weft's packages, and puts them in each app it makes. Warp, the Go module `github.com/flakerimi/weft`, comes from GitHub: an app's `go.mod` requires it at the release's tag, and Go fetches it through its module proxy. An app also needs:

- Go 1.26, which builds its server;
- Node 22.18 or later, which runs the page build's TypeScript directly;
- pnpm 10, which installs Vue, Vite and the other public packages from the npm registry.

`weft create`, `weft dev` and `weft build` check for all three before they write anything.

## Working on Weft itself

You need none of this to build an app. To work on Weft itself, clone the repository, install its packages, and build the themes, which the page build reads:

```sh
git clone git@github.com:flakerimi/weft.git
cd weft
pnpm install
pnpm --filter @weft/themes build
```

A `weft` built from the repository says `0.0.0-dev`. An app it makes gets Warp in `.weft/go`, which its `go.mod` replaces the module with, so working on Weft needs no tag and no network.

To check your setup, build and run this site, which lives in `site-src/`:

```sh
scripts/site.sh
```

It serves the site on `http://localhost:8080`.

## Next

[Create an app](create-an-app.md).
