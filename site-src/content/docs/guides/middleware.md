---
title: Middleware
description: The middleware module - security headers on every answer, one log line per request, and a rate limit per client on the API.
order: 28
---

On this page you learn what the `middleware` module, `github.com/flakerimi/weft/warp/middleware`, adds to an app: security headers on every answer, one log line per request, and a rate limit on `/api`. An app `weft create` makes lists `middleware.Module()` first in `app/modules.go`, so the lines the other modules log at their start, the owner's setup link included, are in its format. It needs no other module.

## Security headers

Every answer, pages, files, API answers and 404s alike, has:

```text
X-Content-Type-Options: nosniff
Referrer-Policy: same-origin
X-Frame-Options: DENY
Content-Security-Policy: frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'
Cross-Origin-Opener-Policy: same-origin
```

and, over HTTPS (`app.IsHTTPS`, which reads `X-Forwarded-Proto` only with `TRUST_PROXY=on`), `Strict-Transport-Security: max-age=31536000`. Each is set before the handler runs, so a handler can replace one. The policy has no `script-src`: the page build writes small inline scripts, which a policy would have to name by their hashes ([Coming next](../coming-next.md)).

## The request log

`LOG` in `.env` is `text` (the default), `json` or `off`. Each request is one line, at info, or at error for a 5xx:

```text
time=2026-11-28T04:00:25.096+01:00 level=INFO msg=request method=GET path=/setup status=200 bytes=3997 ms=0.04 ip=::1
```

The path has no query string, which can hold search text. `ip` is the client's address, as [`ClientIP`](warp.md#behind-a-proxy) reads it. At start the module also makes Go's `slog` default handler, and so the standard logger, write in the same format to the same place, so every line the app writes is one format; `LOG=off` turns off the request log and leaves the app's other lines as they were. `middleware.Module(middleware.LogTo(w))` writes to `w` rather than standard error. When the app stops, the default goes back as it was.

## The rate limit

`RATE_LIMIT` is how many requests to `/api` each client may make, as `300/m` (the default), `20/s` or `5000/h`, from 1 to 1,000,000, or `off`. A client is [`ClientKey`](warp.md#behind-a-proxy): an IPv4 address, or the /64 an IPv6 one is in. Each has a bucket that holds n and fills again at n a period; a request it refuses is 429 `rate_limited`, `Too many requests; wait a moment and try again.`, with `Retry-After` in whole seconds until one is allowed. At most 50,000 clients are remembered, the one seen longest ago dropped first. Pages and files are never limited: they come from memory and cost nothing to answer.

A mistake stops the app, with the `LOG` and `RATE_LIMIT` ones reported together: `RATE_LIMIT is "lots"; set it like 300/m (requests a minute), 20/s or 5000/h, or off.`, `LOG is "xml"; set it to text, json or off in .env.`

## How it wraps the app

The headers and the log go around every request with `app.WrapAll`, and the rate limit around the API alone with `app.Wrap`; [Warp](warp.md#plain-handlers-and-middleware) says how the two differ. Behind a proxy, set `TRUST_PROXY=on`, or every client is the proxy, which shares one bucket and is logged as one address.
