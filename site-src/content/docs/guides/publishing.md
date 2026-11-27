---
title: Publishing
description: Build one binary that holds your app with weft build, and run it on a server or in a container.
order: 31
---

On this page you build your app into one binary and run it on a server. The binary holds the pages, the API and the web server. It needs no Node and no files beside it, unless you serve some from disk.

## Build

In the app's folder:

```sh
weft build
```

It builds the pages into `web/dist`, then the Go binary, which embeds them, and names it after the app: `myapp` for an app made with `weft create myapp`. `-o <file>` writes it elsewhere. The binary embeds `web/dist` as the page build left it, which is why `weft build` always builds the pages first.

`weft build` builds the binary with:

- `-trimpath`, which leaves your machine's paths out of it;
- `-ldflags "-s -w"`, which leaves out the debug tables, so it is smaller;
- `CGO_ENABLED=0`, which makes a static binary that runs on a bare Linux image. When your environment sets `CGO_ENABLED`, its value is used instead.

## For another system

`GOOS` and `GOARCH` pass through to Go. For a Linux server on arm64:

```sh
GOOS=linux GOARCH=arm64 weft build -o myapp-linux-arm64
```

## Run

Copy the binary to the server and start it. It listens on port 8080 on every interface; set `PORT` and `HOST` in the environment, or in a `.env` file in the folder it runs in, to change that (see [Configuration](warp.md#configuration)). Put it behind your server's proxy for TLS. Every page and file is in memory from the start, already compressed, so a request costs a map lookup and a write.

On SIGTERM or Ctrl-C it stops taking requests, lets the ones in flight finish for up to 10 seconds, stops its modules, and exits with status 0.

## In a container

The binary runs on an empty image. Build it for Linux, then copy only the binary into the image:

```sh
GOOS=linux GOARCH=amd64 weft build -o myapp
```

```dockerfile
FROM scratch
COPY myapp /myapp
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/myapp"]
```

Build the image from the app's folder, after `weft build`. A `.dockerignore` with the two lines `*` and `!myapp` keeps the build context to the binary alone. Build for the architecture the image runs on: `GOARCH=arm64` for an arm64 host.

## How this site is published

`scripts/publish-site.sh <version>` publishes weft.base.al. It:

1. builds a release of the `weft` command for four platforms, with `scripts/release.sh`;
2. builds this site with `scripts/site.sh build`, with `WEFT_SITE_VERSION` set to the version, which `/index.md` names (a build without it names no version);
3. makes a tree with the Go sources, the built pages, a Dockerfile that compiles the site's binary in the image and copies it onto `scratch`, the installer, every release `site/releases.txt` lists as published, and the new one;
4. pushes the tree to the server, which builds the image and runs it, lists the version in `site/releases.txt` as soon as the push succeeds, so every later publish keeps it, and waits until the new version answers.

A version that is already released, listed or in `site/releases`, is refused: building it again would change tarballs its users have checksums for. The release is built in a temporary folder and copied into `site/releases` only once it is pushed, so a publish that fails earlier leaves `site/releases` as it was and can be run again with the same version. `--dry-run` builds the same way and stops before the push.

The releases are not in the binary. They are in the image under `/data/releases`, served with [`warp.WithFiles`](go-serving.md#files-from-disk), so the binary does not grow with each release. `scripts/rollback-site.sh <version>` pushes an earlier version's tree again.

## Page weight

This site holds each docs page to 14 KB brotli, about what a browser receives in the first round trip on a new connection. That counts the HTML, its CSS, the loader every page with an island starts with (`boot-*.js`, about 0.5 KB), and the scripts of the page's own islands with the chunks they import. It does not count the shared Vue runtime, `client-*.js`, which a browser caches once for the whole site and fetches only when an island starts, nor fonts, which are cached and preloaded. `site-src/main_test.go` checks every docs page after each build.
