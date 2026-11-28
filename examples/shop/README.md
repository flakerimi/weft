# shop

The example app of the Warp slice: a small back office for the products a
shop sells. It is shaped as `weft create` shapes an app, and uses what a
developer would:

- `app/modules.go` lists `middleware` (security headers, a request log and a
  rate limit), `data` (SQLite, through `warp/data`), `mail` (printed to the
  log), `auth` (a login with four roles, invitations and password reset,
  `warp/auth`) and `products`;
- `app/products/` is one model and one line of routes, `data.Resource`, with a
  permission per action, and `products_test.go`, the whole story over HTTP:
  the owner's, and the one with a member, a viewer and an API key;
- `web/` calls the server only through `web/.weft-types/api.ts`, which
  `weft gen` writes from the Go types: a login page, the owner's setup page,
  pages to ask for and use a mailed link, `/account`, `/users`, `/products`
  (a table with search, a status filter, and a dialog to make, change and
  delete, each shown by permission) and `/products/<id>`, a page built once
  and served for every id.

Run it from the repository's root with this repository's Weft:

```sh
scripts/shop.sh          # weft dev, on http://localhost:5173 (SHOP_PORT=5180 for another)
scripts/shop.sh build    # weft build, which writes examples/shop/shop
scripts/shop.sh test     # the Go tests, the web tests (pnpm test) and the type check
```

The script writes `.env` when there is none, with a fresh `AUTH_SECRET` and
`AUTH_OWNER_EMAIL=owner@example.com`, and `API_KEYS=on`; `.env.sample` says
what each value is.
At the first start the log says `Set up the owner at …/setup#token=…`: open
that link to choose the owner's password. Everything a run makes here
(`.weft/`, `node_modules/`, `web/dist/`, `web/.weft-types/`, `data/`, `.env`,
`pnpm-lock.yaml`, `shop`) is git-ignored.

The suite drives a copy of this folder: `cmd/weft/shop_test.go` builds it,
runs its Go and web tests and its type check, requests its pages and API,
and, when Chrome or Chromium is installed, drives it in the browser with
`test/browser.mjs`.
