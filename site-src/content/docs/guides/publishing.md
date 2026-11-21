---
title: Publishing
description: Build one binary that holds your site, and run it on a server.
order: 25
---

On this page you build your site into one binary and run it on a server. The binary holds the pages, the API and the web server. It needs no Node and no files beside it, unless you serve some from disk.

## Build

Build the pages, then the binary:

```sh
pnpm --filter hello build
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o hello-site ./hello
```

- `CGO_ENABLED=0` makes a static binary, which runs on a bare Linux image.
- `-trimpath` leaves your machine's paths out of it.
- `-s -w` leaves out the debug tables, so it is smaller.

Build for another system with `GOOS` and `GOARCH`. For a Linux server on arm64:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o hello-site ./hello
```

Always build the pages first. The binary embeds `dist/` as it is when `go build` runs.

## Run

Copy the binary to the server and start it. The `hello` program listens on port 8080; put it behind your server's proxy for TLS, or listen on another port. Every page and file is in memory from the start, already compressed, so a request costs a map lookup and a write.

## In a container

The binary runs on an empty image. This Dockerfile builds it and copies only the binary into the final image:

```dockerfile
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/hello-site ./hello

FROM scratch
COPY --from=build /out/hello-site /hello-site
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/hello-site"]
```

Build it from the repository's root. Build the pages before `docker build`, so `hello/dist` is in the context, and put `**/node_modules` in a `.dockerignore`, so the context stays small.

## How this site is published

`scripts/publish-site.sh <version>` publishes weft.base.al. It:

1. builds a release of the `weft` command for four platforms, with `scripts/release.sh`;
2. builds this site with `scripts/site.sh build`;
3. makes a tree with the Go sources, the built pages, a Dockerfile like the one above, the installer, every release `site/releases.txt` lists as published, and the new one;
4. pushes the tree to the server, which builds the image and runs it, lists the version in `site/releases.txt` as soon as the push succeeds, so every later publish keeps it, and waits until the new version answers.

A version that is already released, listed or in `site/releases`, is refused: building it again would change tarballs its users have checksums for. The release is built in a temporary folder and copied into `site/releases` only once it is pushed, so a publish that fails earlier leaves `site/releases` as it was and can be run again with the same version. `--dry-run` builds the same way and stops before the push.

The releases are not in the binary. They are in the image under `/data/releases`, served with [`weft.WithFiles`](go-serving.md#files-from-disk), so the binary does not grow with each release. `scripts/rollback-site.sh <version>` pushes an earlier version's tree again.

## Page weight

This site holds each docs page to 10 KB brotli. That counts the HTML, its CSS, the loader every page with an island starts with (`boot-*.js`, about 0.5 KB), and the scripts of the page's own islands with the chunks they import. It does not count the shared Vue runtime, `client-*.js`, which a browser caches once for the whole site and fetches only when an island starts, nor fonts, which are cached and preloaded. `site-src/main_test.go` checks every docs page after each build.
