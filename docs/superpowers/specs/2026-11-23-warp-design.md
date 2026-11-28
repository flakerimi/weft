# Warp: design

Date: 2026-11-23
Extends `2026-11-20-weft-design.md`. It replaces that design's *Go side*, *The Go-to-Vue contract* and *Login and permissions* sections, and its phases 4b and 5.

## Intent

Weft is the UI. **Warp** is the Go side: the warp is strung on the loom first and holds everything, and the weft is woven across it. Together they are one full-stack thing: `weft create` makes an app whose Go half is a set of Warp modules and whose Vue half is Weft pages, built into one binary.

Warp takes the features of the owner's Base framework (`base-go/base-core`, with its CLI `base-go/cmd`) and rebuilds them. It takes the logic and the feature list, not the code, and it does not depend on Base. Base is API only; Warp is the API half of an app that also has a UI, so each feature is designed with its Vue side.

Success: `weft create shop`, `weft scaffold product name:string price:float cover:image category:belongs_to`, set a few values in `.env`, and there is a working app with login, a products table with create, edit and delete, uploads, and typed API calls in Vue, without the developer writing an HTTP handler or a fetch call.

## Decided by the owner (2026-11-23)

- The Go side is named Warp; the UI stays Weft; the command stays `weft`.
- Base's features come into Warp as logic, rebuilt in a modern way. No code is copied and Base is not a dependency.
- **The core is dependency-free:** standard library only.
- **Go types are the TypeScript types** the Vue side uses for API calls.
- **Warp is modular like Base:** one package per feature, so it grows and stays readable.
- **Built-in modules are configured through `.env`.**
- **`weft scaffold`** generates a resource from a name and typed fields, like Rails and like `base g`.

## What was learned from Base

Both repositories were read on 2026-11-23 (`base-core` main at `073c51b`, `cmd` at `ccc8309`, release v2.2.2). The findings are the reason for several rules below.

Kept from Base, because it works and its users know it:

- The module shape: a model, a service, a controller and a module file per resource, registered in one place.
- The generator's field vocabulary and its naming (snake_case JSON, kebab-case plural routes under `/api`).
- The built-in list: authentication, authorization with roles, profile, media and attachments, email, events, scheduler, translation, OAuth, API keys, rate limiting, logging.
- Routes for select options (`/all`, id and name only).
- First registered user becomes the owner.

Changed, because the code does not do what its README says:

| In Base today | In Warp |
| --- | --- |
| Validation tags are declared and never enforced; there are no per-field errors | Every typed input is validated before its handler runs; failures answer 422 with an error per field |
| `Can()`, `HasRole()` and the role middleware always answer 500, and `HasPermission` always returns true | Permission checks are real, tested, and required on every scaffolded route |
| Requiring a login is one global switch, off by default | Each route says whether it needs a session and which permission; scaffolded routes need one by default |
| The token is a bearer JWT that the page must store | The session is a signed HttpOnly cookie; a bearer token is for other clients |
| Lists have no search, no filter, and silently ignore an unknown sort field | Lists take `q` and `filter[field]`, and an unknown field is a 400 |
| An update cannot clear a field | Updates are PATCH with present-or-absent fields |
| Generated uploads always answer 500; file URLs are absolute to the server's own port | Attachments work from the scaffold; URLs are paths under `/api/files/` |
| The emitter panics on first use and is not connected to the socket | Events are connected to server-sent events, which `useLive` reads |
| Three different response envelopes | One success shape and one error shape |
| Swagger 2.0 with list items typed `any` | Types generated from the Go types themselves; OpenAPI 3.1 from the same description |
| `start` does not reload | `weft dev` already rebuilds and restarts Go on a change |
| No tests | Every module is written test first |

## Rules

1. **Standard library only, in all of Warp.** This holds for the data layer too: it is written on `database/sql`, which is the standard library. The one third-party thing a database needs is its driver, and the *app* imports that, in its own `go.mod`, the way `database/sql` intends. No ORM.
2. **A feature that cannot be done with the standard library is an adapter outside Warp**, in its own Go module under `adapters/`, which an app adds to its `go.mod` when it wants it. Expected: a Redis cache, perhaps an S3 client if signing requests by hand proves unwise.
3. **One declaration.** A record, a request or a response is a Go struct, declared once. Its JSON names, its validation rules, its database columns, its TypeScript type and its form schema all come from that struct and its tags.
4. **One module per feature**, and a module never imports another module's internals: it uses what the other exports, or an interface in the core.
5. **A module that is off costs nothing**: no routes, no tables, no startup work.
6. **Errors are one sentence** that names the thing and says what to do, at startup, at build and in API answers.
7. **Every module is test first**, and every scaffolded resource comes with tests that pass.

## Architecture

```
github.com/flakerimi/weft
  warp/                  the core: App, Module, typed routes, errors, validation, env, Serve
  warp/data/             records in a database, on database/sql
  warp/auth/             sessions, users, roles, permissions
  warp/storage/          attachments
  warp/mail/             email
  warp/events/           the emitter
  warp/live/             server-sent events
  warp/jobs/             scheduled work
  warp/i18n/             translated fields
  warp/middleware/       logging, security headers, CORS, rate limit, API keys
  warp/describe/         the app's routes and types as data, for weft gen and OpenAPI
  adapters/<name>/       separate Go modules for third-party needs
  cmd/weft/              the command
```

`weft.Serve` moves to `warp.Serve`, so all Go code has one name. The root package keeps `Version`.

An app:

```
shop/
  main.go                builds the warp.App, lists the modules, serves
  .env, .env.sample      configuration
  app/
    modules.go           the one list of the app's modules; weft scaffold adds a line
    products/
      model.go           the Product struct
      routes.go          its routes
      products_test.go
  web/                   the Weft pages
    pages/products/index.vue
    .weft-types/api.ts   generated: the types and a typed function per route
```

### The core

```go
server := warp.New(warp.FromEnv())
server.Use(app.Modules()...) // auth.Module(), storage.Module(), products.Module(), listed in app/modules.go
if err := server.Serve(dist); err != nil {
	log.Fatal(err)
}
```

- **A module** is `Name() string` and `Register(app *warp.App) error`. Optional interfaces add a step: `Migrator` (tables), `Starter` and `Stopper` (background work). The app starts modules in the order listed and stops them in reverse.
- **Routing is the standard library's `http.ServeMux`** (method and wildcard patterns). Warp writes no router.
- **A route is a typed function:**

  ```go
  warp.Get(app, "/products/{id}", show)        // func(ctx context.Context, in ShowIn) (Product, error)
  warp.Post(app, "/products", create, auth.Can("products:create"))
  ```

  The input struct is filled from the path, the query and the JSON body by tag, validated, and handed to the function. The return value is encoded as JSON. A plain `http.Handler` can still be mounted for anything unusual.
- **Because routes are typed and registered, the app can describe itself.** `warp/describe` lists every route with its method, path, input type, output type and required permission. That one description feeds the TypeScript generator, the OpenAPI document and `weft routes`.

### Wire format

JSON names are snake_case, as in Base. (Claude's call: it matches the database columns and the owner's existing Base apps.)

- **One record:** the object itself.
- **A list:** `{ "items": [...], "total": 42, "page": 1, "per_page": 20 }`. Query: `page`, `per_page` (default 20, at most 100), `sort` (`name`, `-created_at`), `q`, `filter[field]=value`. A field that cannot be sorted or filtered is a 400 that names it.
- **Create** answers 201 with the record; **update** is PATCH and answers the record; **delete** answers 204.
- **An error:** `{ "error": { "code": "validation", "message": "2 fields need attention.", "fields": { "email": "Enter an email address." } } }`. Statuses: 400 malformed, 401 no session, 403 not permitted, 404, 409 conflict, 422 validation, 429, 500.

### Validation

Rules are in the struct's `warp` tag: `warp:"required,max=120"`, `email`, `url`, `min=`, `enum=basic|pro`, `unique`. They run on every typed input before the handler. The same rules are written into the generated form schema, so `Form` checks them in the browser and shows the server's answer on the same field when the two disagree.

### Go types are the TypeScript types

`weft gen` (and `weft dev`, on each Go change) builds the app, asks it for its description, and writes `web/.weft-types/api.ts`:

```ts
export interface Product { id: number; name: string; price: number; cover: Attachment | null; created_at: string }
export const api = {
  products: {
    list(query?: ListQuery<Product>): Promise<List<Product>>,
    show(id: number): Promise<Product>,
    create(input: ProductCreate): Promise<Product>,
    update(id: number, input: Partial<ProductCreate>): Promise<Product>,
    remove(id: number): Promise<void>,
  },
}
export const schema = { product: { /* label, kind, rules, options per field */ } }
```

A page calls `api.products.list({ sort: '-created_at' })`. A renamed Go field is a type error at every call site after the next generate. The file is never edited by hand and is not committed.

### Data

`warp/data` stores structs through `database/sql`:

- `data.Store[T]` gives `List(ctx, Query)`, `Get`, `Create`, `Update`, `Delete` for a struct, with the columns, the sortable and filterable fields and the searchable text fields read from its tags. `warp.Resource(app, "/products", store, ...)` registers the five routes and `/all` on it.
- Soft delete (`deleted_at`) and timestamps as in Base.
- Relations: `belongs_to` (the id and the related record's id and name in answers, in lists too), `has_many`, `many_to_many` (ids in and out).
- **Migrations are additive and automatic:** at start, a missing table or column is created. Anything that loses data (a dropped or retyped column) is never automatic: the app refuses to start and says what differs. `weft db` shows the difference.
- **Engines:** SQLite first (the default in a new app, through a pure-Go driver so no C compiler is needed), then PostgreSQL, then MySQL (the owner, 2026-11-23). The driver is the app's dependency. The differences between the three (placeholders, quoting, column types, returning the new id) are in one small dialect per engine, and every `Store` test runs against all three.
- Custom queries use `database/sql` directly; `data` exposes the `*sql.DB`.

### Built-in modules

Each is switched on and configured in `.env`. `weft create` writes `.env.sample` with every value and a comment, and `.env` with working development values. At start, a module that is on and misconfigured stops the app with one sentence per missing value.

| Module | `.env` | What it gives |
| --- | --- | --- |
| `auth` | `AUTH=on`, `AUTH_SECRET`, `AUTH_SESSION_DAYS`, `AUTH_REGISTRATION=open\|invite\|closed`, `API_KEYS=on` | Register, login, logout, current user, forgot and reset password, change password, profile and avatar, API keys for other programs (moved here from `middleware`, 0.2.0). Sessions kept in a table, named by a signed HttpOnly cookie. Roles Owner, Administrator, Member, Viewer, each including the next one's permissions; permissions named `resource:action`; the first user is the Owner. `auth.Can("products:edit")` on a route. Vue: `useSession()`, a login page, `perm` on `Button` and row actions, a route guard. |
| `storage` | `STORAGE=local`, `STORAGE_DIR`, `STORAGE_MAX_MB` | Attachments on a record's `image` and `file` fields, with type and size limits per kind, served under `/api/files/`. Vue: `FileUpload` bound to the field. |
| `mail` | `MAIL=smtp\|log`, `MAIL_HOST`, `MAIL_PORT`, `MAIL_USER`, `MAIL_PASSWORD`, `MAIL_FROM` | Send text and HTML email from templates. `log` prints the message, for development. Password reset uses it. |
| `events` | none | `events.On("products.created", fn)` and `Emit`. `data.Store` emits `<resource>.created`, `.updated`, `.deleted`. |
| `live` | `LIVE=on` | Server-sent events at `/api/live`, by topic, checked against the session's permissions. An event can be published to a topic in one line, so a table refreshes when a record changes. Vue: `useLive`, `useLiveSeries`. |
| `jobs` | `JOBS=on` | Work on a schedule (`every 5m`, `daily 03:00`), with a list of jobs and their last result. |
| `i18n` | `LOCALES=en,sq` | Translated fields on a record. |
| `middleware` | `LOG=json\|text`, `CORS_ORIGINS`, `RATE_LIMIT` | Request logging on `log/slog`, security headers, CORS, a rate limit that works. API keys are `auth`'s (0.2.0). |
| `docs` | `API_DOCS=on` | An OpenAPI 3.1 document at `/api/openapi.json`, from the description. |

Later, as adapters or modules when a product needs them: OAuth sign-in (Google, Apple), S3 storage, a Redis cache, email through provider APIs.

### `weft scaffold`

```sh
weft scaffold product name:string price:float description:text published:bool \
    cover:image category:belongs_to tags:many_to_many status:enum:draft|live
```

writes `app/products/` (model, routes, tests), adds the module to `app/modules.go`, regenerates the types, and writes `web/pages/products/index.vue`: a table with server-side paging, sorting, search and filters, and a dialog to create and edit from the generated schema. It refuses, writing nothing, if the resource exists.

Field types, from Base's generator: `string`, `text`, `email`, `url`, `phone`, `password`, `int`, `float`, `bool`, `date`, `datetime`, `json`, `image`, `file`, `translated`, `belongs_to`, `has_many`, `many_to_many`, and `enum:a|b|c`. A name with no type is inferred as Base does (`email`, `*_at`, `is_*`, `*_id`).

`weft scaffold --api` writes only the Go half. `weft routes` prints every route with its permission. Removing a resource is done by hand: the command lists the files and the line to remove, and deletes nothing.

## Phases

These replace phases 4b and 5 of the Weft design. Each gets its own plan. W2 to W4 were first built together as one thin slice (`2026-11-25-warp-slice-design.md`, plan `docs/superpowers/plans/2026-11-25-warp-slice.md`), which each phase now widens.

| Phase | Contents |
| --- | --- |
| W1 | Delivered (plan `docs/superpowers/plans/2026-11-25-warp-core.md`). The core: `warp.App`, modules, env configuration, typed routes on `ServeMux`, the error shape, validation; `weft.Serve` becomes `warp.Serve`; the template's `main.go` and `app/` move to this shape |
| W2 | A first pass delivered by the slice (`2026-11-25-warp-slice-design.md`): `Describe` in the core rather than `warp/describe`, `weft gen`, `weft routes`, `weft dev` regenerating. Left: an `enum` typed as its values, a typed `useParams`, an API on another origin. As designed: `warp/describe`, `weft gen` (types, the typed client, form schemas), `weft routes`; `weft dev` regenerates on a Go change |
| W3 | A first pass delivered by the slice: `warp/data` with SQLite, `Store`, additive migrations, list queries, and `data.Resource` in place of `warp.Resource`. Left: PostgreSQL, MySQL, relations, `weft db`. As designed: `warp/data`: `Store`, additive migrations, list queries, `warp.Resource`; SQLite, then PostgreSQL, then MySQL |
| W4 | Delivered in 0.2.0, "accounts" (plan `docs/superpowers/plans/2026-11-28-accounts.md`), after a first pass by the slice: four roles and permissions, sessions in a table, invitations, sign-up and reset through `mail`, API keys, `useSession`, `perm` and `Guard` in Vue, and the template's account pages. Left: an avatar (with `storage`, W6) and changing an email. As designed: `auth`: sessions, users, roles, permissions; `useSession`, the login page, `perm`, the route guard; `mail` with the `log` and `smtp` drivers |
| W5 | Not started; the slice's shop is the code it is to be written from. As designed: `weft scaffold`, with `DataTable` and `EntityDialog` (from the Weft design's phase 5) |
| W6 | `storage` and attachments; `events` and `live` with `useLive` |
| W7 | `jobs`, `i18n`, `docs`; `middleware`'s headers, request log and rate limit were brought forward into 0.2.0, and CORS and a `script-src` policy are left |

W1 starts after phase 4 (app tooling) lands, because it changes the template that phase writes.

## Non-goals

- Compatibility with Base's wire format or its apps. Warp is for new apps.
- An ORM, or a query builder beyond what `Store` needs.
- A chat-room WebSocket. Live data is server-sent events.
- Copying Base's code. Base has no licence file; nothing is taken from it.

## Calls made by Claude (open to correction)

- The data layer is on `database/sql` with no ORM, so all of Warp, not only its core, stays on the standard library.
- Routes are typed functions on the standard library's `ServeMux`; Warp has no router and no context type of its own.
- TypeScript types come from the running app's own description of its routes, not from parsing Go source or from Swagger comments.
- snake_case JSON; the list shape `{items, total, page, per_page}`; PATCH for updates; 422 for validation.
- Sessions in a signed HttpOnly cookie, with a bearer token only for clients that are not the app's own pages.
- Migrations add and never remove.
- SQLite by default through a pure-Go driver.
- Modules are listed explicitly in `app/modules.go`; nothing registers itself from an `init` function.
- Scaffolding never deletes: there is no `destroy` command.

Made while planning and building W1 (2026-11-25):

- No `weft.Serve` is left behind: the root package keeps only `Version`. Nothing is published, so a forwarding function would only be a second name.
- `warp.Serve(dist, api, opts...)` is the handler, as `weft.Serve` was; `app.Serve(dist)` builds it over the app's own API, listens, and stops cleanly on SIGINT or SIGTERM. The template calls `log.Fatal` only when `Serve` returns an error, since a clean stop returns nil.
- Mistakes in registration are collected, not panicked: `app.Use`, `app.Handle` and `warp.Get` return nothing, and `app.Err()` and `app.Serve` report every mistake, one sentence each. `warp.Get(app, ...)` has no error to check, and a panic at import time has no place to name the file.
- Route paths are written without `/api`, which the app adds: `warp.Get(app, "/products/{id}", show)` answers at `/api/products/{id}`.
- Every exported field of an input struct has exactly one `json`, `query` or `path` tag. A field without one is a registration mistake, so no field gets Go's default name in a snake_case API.
- Struct types inside a body field are checked at registration too: every exported field needs a usable `json` name, and a `query` or `path` tag inside is a mistake. An input that embeds a pointer to a struct is a mistake, since a body that leaves its fields out would leave it nil and a promoted field would panic.
- An unknown field in a JSON body is a 400. A misspelt field that is silently dropped is how an update loses data.
- Typed routes take JSON bodies only: a body that is not empty must be `application/json` or a `+json` type, else 415 `unsupported_media_type`. A page on another site can send a form or `text/plain` with the visitor's cookies and no preflight, and sessions in cookies come in W4. This keeps another site from sending a route a body; it does not protect a route that takes no body (see the W4 note below). A body that is JSON but not an object is a 400.
- `POST` answers 201 unless the route says `warp.Status(200)`; an output of `warp.None` answers 204.
- A path value that does not convert is a 404, not a 400: `/api/products/abc` names nothing.
- `required` on a pointer passes whenever the pointer is not nil: a pointer is how an input says "present, and it may be zero". Every other rule passes an empty value, except behind a pointer that is not nil (see the fix wave below).
- `unique` is parsed in W1 and enforced in W3, by `warp/data`, which is the first thing that can look.
- `weft dev` sets `HOST=127.0.0.1`, and `warp` watches for the dev parent, so `dist_dev.go` keeps no logic.
- `weft build` makes the binary to ship (the owner, 2026-11-25: the docs say `weft build`, not `go build`). It builds with `-trimpath -ldflags "-s -w"`, and with `CGO_ENABLED=0` unless the environment sets it; `GOOS` and `GOARCH` come from the environment. No doc page builds a site by hand inside the Weft repository.
- Out of W1: `filter[...]` and list queries (W3), present-or-absent `PATCH` fields (W3), `auth.Can` and a route's permission (W4), `warp/describe` (W2). `Route` is shaped so each can add a field.

Made in W1's final fix wave (2026-11-25):

- What a 422 lists is bounded: at most 100 fields, at most 16 KiB of key bytes in all, counted before JSON escaping (a map key of escaped characters can make the answer up to about 100 KB), and no field whose key is longer than 1 KiB; when one is left out the message is `More fields need attention than are listed; the first N are listed.` (`More fields need attention than can be listed.` when none is). The walk stops once the 100 or the 16 KiB are spent, and lists in a fixed order: struct fields as declared, list entries by index, map entries by sorted key. A typed route also refuses a body nested more than 64 levels deep, before decoding: 400, `The body is nested more than 64 levels deep.` Before these, a 103 KB body could make a 100 MB answer.
- A rule on a pointer that is not nil applies to what it points at, even a zero: the pointer says the value was sent, so `min=1` on a `*int` holding 0 fails. `required` on it still passes, and a zero not behind a pointer is still empty.
- `warp.Option`, the page-serving option, is `warp.ServeOption`, leaving `Option` free for the app's own options later. No alias is kept.
- `warp.Status` takes 200 to 599: a 1xx is not an answer a route can give.

## Notes from W1 for later phases

- **W2.** `Route` has no name or metadata, and the typed client needs a name per route. The parsed rules are unexported, so `warp/describe` needs them exported. `Out` types are not checked at registration. A wrong-typed value inside a list or map is reported without its index or key (`lines.qty`), which the form schema will need.
- **W3.** A present-or-absent `PATCH` needs the raw body, which `Request(ctx)` no longer has. `float32` fields are compared with their rules after widening, so `float32(0.1)` fails `max=0.1`.
- **W4.** `RouteOption` is a func over an unexported struct, so `auth.Can` in another package can only compose `With`: it needs an exported way to attach a permission to a route. A route that takes no body is open cross-site, so the session cookie must be `SameSite=Lax` or stricter, or state-changing routes need an `Origin` check.
- **W6.** `Shutdown` does not cancel request contexts, so an open SSE stream makes each stop take the full 10 s: it needs a `BaseContext` cancelled at the drain. `Start`'s context ends at the stop signal, before the drain. Shutdown's error is always reported as a timeout.
- **W7.** There is no body-read or idle timeout.
- **Validation's cost, if `MAX_BODY_MB` is raised far above 1.** Validating a map whose entries fail costs about one allocation per failing entry (about 24 MB for 100,000), and a nested map is probed again for each failing map above it. Both are bounded by the body limit.

## Open for the owner

1. **Existing Base apps.** This design does not migrate them. Say if one should be the first Warp app, since that would decide which modules come first.

Answered 2026-11-23: MySQL is the third engine; the defects found in `base-core` and `base-go/cmd` are fixed in those repositories, as separate work that this design does not depend on.

## Made while planning 0.2.0

The calls of `docs/superpowers/plans/2026-11-28-accounts.md`, copied as they stand when its Track E ends; where they and this design differ, these are what was built.

Each was left open by the designs; each is Claude's, open to correction, and is to be copied into the Warp design's calls by Task E5.

- **Call: roles live in code, not a table.** Four, ranked `owner` > `administrator` > `member` > `viewer`, each with the permissions of those below it, as the design names them. An app adds permissions with `auth.Grant(role, permissions...)`. A table of roles is a settings screen nobody asked for, and the design fixes the four.
- **Call: default grants.** `viewer`: `*:read`. `member`: also `*:create`, `*:edit`, `*:delete`. `administrator`: also `users:read`, `users:invite`, `users:edit`, `users:delete`. `owner`: every permission. A `*` resource never covers `users`, which is granted only by name, so a role that edits products cannot edit accounts. The verbs are the ones `auth.Per` already uses.
- **Call: no one changes their own role or removes themselves through the routes** (409 `conflict`, `You cannot change your own role or remove yourself; ask another owner or administrator.`). An actor acts only on a user whose role is not above theirs, and gives only a role not above theirs. With these, an owner always remains, without a rule of its own. `auth.RemoveUser`, the tool function, keeps its slice doc.
- **Call: sessions live in a table `sessions`**; the cookie is `<session id>.<token>.<mac>`: a 32-byte random token in base64url, and HMAC-SHA256 with `AUTH_SECRET` over `<id>.<token>`. The row keeps the token's SHA-256. The mac keeps a forged cookie from costing a database read, and keeps "changing `AUTH_SECRET` signs everyone out" true. The slice's cookies are not read: everyone signs in once after upgrading.
- **Call: a session lasts `AUTH_SESSION_DAYS` from sign-in, not from last use**; `seen_at` is written at most once every 5 minutes per session, so a request does not mean a write; a user has at most 50 sessions, and a sign-in past that ends the one seen longest ago; expired sessions and tokens are removed at `Start` and at each sign-in, so no background goroutine is needed.
- **Call: `auth` needs `mail`.** Password reset is part of the login and always mails, and the `log` driver needs no settings, so there is one shape, `data.Module(), mail.Module(), auth.Module()`, and no route that exists and cannot work. An app made before 0.2.0 gets `warp: module "auth" needs the "mail" module; list mail.Module() before it in app/modules.go.`
- **Call: `AUTH_REGISTRATION` defaults to `invite`.** `open` lets anyone ask for a sign-up link; `invite` takes only invitations; `closed` takes neither (users come from `auth.AddUser` and the setup link). A new app should not take strangers by default.
- **Call: an open registration makes a `viewer`.** The least role; an owner raises it. (See **Owner** 2.)
- **Call: registration, invitation and reset are one mechanism:** a mailed link `<APP_URL>/password#token=<token>` to one page that sets a password, signs in and ends the token. `register` makes no user until the link is used, so an address is proved before an account exists.
- **Call: token lifetimes.** Reset 1 hour; registration 24 hours; invitation 7 days. A new reset or registration for an address ends that address's earlier one of that purpose; a new invitation for an address replaces the earlier one. A token is 32 random bytes, base64url, kept as its SHA-256, used once.
- **Call: `register` for an address with an account mails a reset link** ("You already have an account"), and `forgot` for an address with none mails nothing; both answer 204 with no body (see Call (A5) below), so the answer never says which.
- **Call: mail is sent from a queue in memory**, so `register` and `forgot` answer at once whatever the mail server does: 100 messages; one sender; each try has 30 s; three tries, 5 s and then 30 s apart; then one error line naming the recipient and the subject, never the body. A full queue answers 503 `The server is busy sending mail; try again in a moment.`
- **Call: SMTP security comes from the port.** `MAIL_PORT=465` is TLS from the first byte; any other port must offer STARTTLS, except on a loopback host (a development relay); `MAIL_PORT` defaults to 587. No `MAIL_TLS` value, so the design's six variables stay the whole list. Start does not connect: an app starts while its mail server is down, and the queue's error line says so.
- **Call: `MAIL` unset is `log`**, with one line at start, `mail is printed, not sent; set MAIL=smtp in .env to send it.` The log driver keeps the last 20 messages for `mailtest`.
- **Call: the per-account count is 10 failed logins in 10 minutes for one email, across every address**, beside the slice's 5 a minute per email and address and 30 a minute per address. A completed reset clears it; a successful login does not, since otherwise someone who knows the password could reset the count for a guesser elsewhere, and the count ends with its window anyway. A successful login gives back its own try, so only failures count (as fixed in review: twenty good sign-ins in a row all succeed). Counted for every email, with an account or without, so it leaks nothing. (See **Owner** 4.)
- **Call: mail-sending limits.** `register` and `forgot`: 3 an hour per address mailed, 10 an hour per client (`ClientKey`); `invite`: 50 an hour per inviter. `token` and `password`: 30 a minute per client. Every limiter keeps at most 10,000 keys, the oldest dropped, as the slice's do.
- **Call: API keys belong to `auth`, not `middleware`.** A key acts as a user, and the guard that checks permissions is auth's; the design's table put API keys under `middleware`, whose other parts know no users. The variable keeps its name, `API_KEYS=on`, read by auth. A key is `Authorization: Bearer wk_<id>_<secret>`, the secret 32 random bytes in base64url, kept as its SHA-256; it has a name (60 characters), a role no higher than its maker's, and an expiry of 1 to 365 days (default 90); it acts with the lower of its role and its user's role now; a user has at most 20; `seen_at` is written at most once every 5 minutes.
- **Call: a key cannot reach `/api/auth` except `me` and the permission-guarded routes.** A stolen key cannot make more keys, read sessions or change a password.
- **Call: the security headers**, on every answer when `middleware` is listed: `X-Content-Type-Options: nosniff`; `Referrer-Policy: same-origin`; `X-Frame-Options: DENY`; `Content-Security-Policy: frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'`; `Cross-Origin-Opener-Policy: same-origin`; and over HTTPS (`app.IsHTTPS`) `Strict-Transport-Security: max-age=31536000`. Each is set before the handler runs, so a handler can replace one. No `script-src`: the page build writes inline scripts (`data-weft-prepaint`, speculation rules) that a policy would need the build's hashes for (see **Owner** 3).
- **Call: `RATE_LIMIT` is `<n>/<s|m|h>` per client (`ClientKey`), on `/api` only**, default `300/m`, `off` to turn it off: a bucket per client that holds n and refills n per period; a refused request is 429 `rate_limited`, `Too many requests; wait a moment and try again.`, with `Retry-After` in whole seconds until one request is allowed; at most 50,000 clients remembered, the oldest dropped. Pages come from memory and cost nothing to answer.
- **Call: `LOG` is `text` (default), `json` or `off`.** One line per request, pages and API: method, path without the query (a query can hold search text), status, bytes, milliseconds and `ClientIP`, at info, or error for a 5xx. `middleware`'s `Start` also makes slog's default handler that format, so every line Warp writes, the setup link included, is one format.
- **Call: CORS stays out of 0.2.0.** Cookies, cross-site protection and `TRUSTED_ORIGINS` already decide which sites may send; the generated client takes no other origin; CORS stays with W7 in `coming-next.md`.
- **Call: the server's timeouts are fixed:** `ReadHeaderTimeout` 10 s (as now), `ReadTimeout` 30 s (headers and body; at the default 1 MiB limit a body must arrive at 35 KB/s), `IdleTimeout` 120 s; no `WriteTimeout`, since a server-sent stream (0.4.0) writes for as long as it is open. Uploads (0.4.0) will lengthen the read per route with `http.ResponseController`.
- **Call: `WrapAll` is a new core method** beside `Wrap`: `Wrap` stays around `/api` alone (the rate limit uses it); `WrapAll` is around pages, files and API (headers and the log use it). `Handler()` (the API alone) is inside every `WrapAll` too, so a test through `warptest` sees the headers.
- **Call: a date is its own rule, `date`,** on a `time.Time` or `*time.Time`: the value is midnight UTC (`Send a date alone, such as 2026-03-29T00:00:00Z.`). A date has no zone; it is shown and entered as written. A datetime is shown and entered in the browser's time zone and sent in UTC.
- **Call: `weft scaffold` writes a `date` or `datetime` field that is not `required` as a pointer**, as `:nullable` does, so an emptied control clears it instead of sending year 1.
- **Call: `perm` hides; it does not disable.** A `Button` with `perm` renders nothing until the session is known and has the permission, on the server always; `DataTable`'s `perm="products"` does the same for New (`products:create`), Edit (`products:edit`) and Delete (`products:delete`), and drops the Actions column when neither row action is left. For the role `owner`, `can()` is always true, as the server's guard is.
- **Call: the route guard is a component and a function,** `<Guard perm="…">` and `requireSession({ perm })`, since pages are documents and there is no client router until 0.5.0; the router will call the same function.
- **Call: `nextPath` moves into `@weft/ui`**, beside `useSession`, since the guard writes `next` and the login reads it.
- **Call (C1): `refresh()` keeps what is shown until the new answer comes.** `loading` is true only until the first answer, so a `perm` button does not blink out while a page asks again; when answers cross, only the newest question's answer is kept. A 200 that is not a user (no numeric `id`, string `role` and `permissions` list) is the problem `The server could not answer right now; try again in a moment.`, never a user. `signInAgain(signIn)` adds `next` with `&` when `signIn` already has a query; `next` is the path and query, without the fragment, as the template's was.
- **Call (C2): `Button` and `DataTable` load the session on demand** (`src/session/can.ts`), not with the page: `src/session/load.ts`, which brings `session.ts`, is imported with `import()` the first time a mounted component with `perm` checks one (a module nothing imports statically, so an app that also imports `useSession` or `Guard` has no module both imported and loaded on demand, which rolldown warns about), so a page whose buttons have no `perm` neither fetches the chunk nor asks `me`. Imported as it is, the session cost every island with a `Button` about 0.6 KB and broke the page budgets (`/skills`, the landing page's copy button); on demand it costs about 0.18 KB. Each component's check (a ref of `can`) says no until it has mounted and its own `import()` has resolved, so an island that wakes after the session is known still hydrates the server's markup. `scripts/sizes.ts` counts Vite's preload helper with Vue's runtime, where a page build keeps it, rather than in each row.
- **Call (C3): `Guard` follows the session and names its user.** Its default slot is given `{ user }`; after `refresh()` a role that lost `perm` sees `denied`; while a visitor with no session is being sent to sign in it keeps showing `loading`. `denied`'s default is a `warning` Alert, a problem a `danger` Alert with `role="alert"`, `loading`'s a Spinner labelled Loading. It is the group `session`, which the gallery shows after Data; the component counts in `skills/weft-components/SKILL.md` and the docs index are 85, as `catalog.test.ts` asks.
- **Call (C, fix round 1): the session follows the page coming back.** The first question starts two listeners, in `session.ts`, the on-demand chunk, so a page with no session pays nothing: `visibilitychange` to visible and `pageshow` with `persisted` (the back-forward cache) call `refresh()`, so a tab left open after a sign-out elsewhere, or restored after another person signed in, shows who is signed in now. No `BroadcastChannel`: a person acts only in the visible tab, which `visibilitychange` already covers. Whoever waits on an answer that a newer question overtook gets the newest answer; `signOut` leaves `ready` as no one; a Guard whose session ends mid-page sends the visitor to sign in, as on load. The gallery's server answers `GET /api/auth/me`, and only it, with a demo viewer who may edit products (`site-src/main.go`, `packages/vite/test/fixtures/demo-me.json`), so the gallery shows `perm` and `Guard` both ways.
- **Call (C2): no DataTable example for `perm`.** The data-table gallery page had 139 B of its 17 KB budget left, and an example costs about 450 B; the Button example shows `perm` in the gallery, and DataTable's docs and prop say what `perm` does there.

Calls made while building Track A, where this plan left a choice:

- **Call (A2): the default grants count as grants for `Me.Permissions`.** "Every permission named in a `Grant` without a `*`" includes the administrator's `users:read`, `users:invite`, `users:edit` and `users:delete`, so `me` lists them for an administrator and the owner whatever routes exist.
- **Call (A2): `authtest.As` numbers its addresses from the users table**, `n` being the count of users plus one, moving past an address taken after a removal, so authtest keeps no state.
- **Call (A3, as fixed in review): one clock.** `warp/data` exports `(*DB).Now()`, the clock the stores write by, which `data.DB.SetClock` sets; the login reads it for its limits, sessions and mailed links, so a test sets one clock and the login and the stores never disagree. There is no `auth.SetClock`.
- **Call (A3): a spent email-and-address count gives back the account's try**, as a spent account gives back the address's: a try refused before any hash is not a failed login, so one address cannot spend an account's ten with refused tries.
- **Call (A5): `register` and `forgot` answer 204, not 202.** The core answers every `warp.None` route 204 (`warp/app.go`, outside Track A); the point, one answer whatever was found, holds. A core change letting `warp.Status(202)` stand with `warp.None` would make it 202.
- **Call (A5, as fixed in review): `register` and `forgot` look the address up and mail after they answer**, on a goroutine bound to the app (its `Stop` waits for it): at most 32 at once, each for at most 30 s, past which both answer the same 503 for every address. The answer, 204, and the work before it are the same for an address with an account and one without, and whatever the mail module does; a failure is logged, naming neither the address nor the link. A test that reads their mail waits for it.
- **Call (A5): an address has one mailed token at a time.** A new reset or registration ends the address's earlier token of either purpose (`DeleteBy("email")`): an address has an account or has none, so only one purpose can work for it, and the store's list filter takes 200 bytes where an address takes 254.
- **Call (A5): no one signs up before the owner is set up.** `register` answers its 403 while the users table is empty, and `password` refuses a registration token then, since a first user other than the owner would end the setup link.
- **Call (A5): `API_KEYS` is `on`, or `off` or unset**; anything else stops the start: `API_KEYS is "yes"; set it to on or off in .env.`
- **Call (A5): a mailed token is looked up by its hash's unique index**, since the link carries no id; there is no stored value to compare in constant time beyond the lookup.
- **Call (A5): a spent per-address mail count gives back the client's try**, since nothing was mailed; an invitation's try is taken after its checks (role, address, setting), so a refused invitation costs none.
- **Call (A6): a wrong `current` in `change_password` is a 422 on `current`, `This is not your current password.`**, not login's 401, which a page holding a session would read as signed out; it still spends `attempt`'s three counts, so the sixth wrong try from one address is the 429. `update_user` without `role` is a 422 `This is required.` on it, since on PATCH the tag rules run only on what was sent. `end_user_sessions` may be asked of oneself.
- **Call (owner's ruling in review, I5): a password reset ends all of the user's sessions and deletes all their API keys**, in the reset's transaction, since a reset is the way back from a stolen password and a key the thief made would outlive it; **a password change ends the user's other sessions and keeps their API keys**, which are for programs and can each be removed. The login guide says both.
- **Call (A7): `api_keys` keeps `prefix`**, since it holds 4 characters of a secret that is not kept.
- **Call (A7): a key acts with the lower role all through**: `auth.UserOf` gives the handler the user with that role, so `invite`'s and `update_user`'s rank rules hold the key to it, and `me` with a key answers that role and its permissions.
- **Call (A7): the scheme `Bearer` is read in any case** (RFC 9110); another scheme is no key, and the cookie is read.

Calls made while building Tracks B and D (Claude's, in `.worktrees/accounts-b`; the report is `.superpowers/accounts-b-report.md`):

- **Call: the `log` mail driver prints and keeps a message inside `Send`**, not through the queue, so `mailtest.Last` right after a request sees it (and `Last` waits up to 5 s besides, for mail an app sends from a goroutine of its own); the queue, its 100 and its 503 are the `smtp` driver's. The 100 counts messages waiting or being sent, so the 101st is refused whatever the sender is doing.
- **Call: what the sender tries again.** A 4xx answer, a connection that cannot be made or breaks off, and a try that runs out of its 30 s are tried again; a 5xx answer, a server without STARTTLS on a host that is not loopback, and a certificate that does not verify are not. `Send` also refuses an empty Subject (`mail: the message has no Subject; give it one.`) and, after Stop, `mail: Send was called after the app stopped; send while it runs.` The test hooks are `dial`, `tlsRoots` and `wait`.
- **Call: the request log's fields** are `method`, `path`, `status`, `bytes`, `ms` (two decimals) and `ip`, message `request`. `LOG=off` turns off the request log alone and leaves slog's default as it is. `middleware`'s `Stop` gives slog's default to the newest app still running in the process, or, when none is, puts it and the standard logger back as the first `Start` found them, whatever order apps stop in. List `middleware` first in `app/modules.go`, so the lines other modules write at their start are in its format.
- **Call: the rate limit is kept as the generic cell rate algorithm keeps it**, one time per client in whole nanoseconds, so `5/m` answers `Retry-After: 12` exactly; "oldest" is the client seen longest ago. A mistake in `LOG` and one in `RATE_LIMIT` are reported together.
- **Call: a date is written in UTC as well as midnight there**, so `2026-03-29T02:00:00+02:00`, the instant of `2026-03-29T00:00:00Z`, is refused: a date has no zone.
- **Call: `EntityDialog`'s `DatePicker` and `InputTime` come with `useDateFields()`**, called in the island that draws the dialog (a `DataTable`'s too); without it a date is `<Input type="date">` and a datetime `<Input type="datetime-local">`. The page build inlines the CSS of everything a page's islands reach, a dynamic import's included, so drawing them always, even on demand, put their CSS and the calendar's on every page with a `DataTable` (over the gallery's 17 KB budget). `weft scaffold` writes the call for a resource with a time. The date takes the field's `<label for>`; the time is named `labels.timeOf(label)`, `"Starts, time"`.
- **Call: in `EntityDialog`, a date chosen before a time is at midnight** in the browser's zone; a time field starts `null`; a control that answers the value it was given writes nothing, so a datetime's seconds survive an edit of other fields.
- **Call: the time-zone tests run each zone in a `node` of its own**, started with `TZ` set, since V8 keeps the zone a process started with.
- **Call: the record's page `weft scaffold` writes shows a date or datetime with `wireTimeText`**, as its table does.

Calls made while building Track E (Claude's, in `.worktrees/accounts-e`; the report is `.superpowers/accounts-e-report.md`):

- **Call (E, from C's review): a page leaves once.** `session.ts` keeps a `leaving` flag, set by `signOut` and `signInAgain`; `signInAgain`, and so `requireSession`, does nothing while it is set, so signing out on a page with a `Guard` lands on `/login` and several Guards send the visitor once. An answer that brings a user, or the page restored from the back-forward cache, clears it.
- **Call (E1, as changed in the final review's fix round): the 401 refresh is `sessionEnded()`,** exported from `@weft/ui`: `refresh()`, then `requireSession()`. A signed-in page calls it in its `catch` on a 401 (the template's `AccountPanel` through a `failed(e)` of its own) and on a `DataTable`'s `@error`; the login calls nothing, since its 401 is a wrong password. `formProblem` keeps the one argument it had in 0.0.24, so the pages `weft upgrade` adds to an app made then type-check against that app's own `answers.ts`, which upgrade never writes over.
- **Call (E1): `sessionFrom` leaves the template and the shop with `session.ts`**, since nothing else used it.
- **Call (E1): the plan's `TestCreatedAppSignsUpAndResets` is part of `TestCreatedAppIsTheFrame`**, on the same build, to save a second build of minutes: with `AUTH_REGISTRATION=open`, the built binary serves the pages, sets up the owner, signs a stranger up and resets a password through the mail it prints.
- **Call (E2): `/users` uses `DataTable`'s own dialog for a role**, with `schema.RoleInput` and `perm="users"`, rather than a `Select` in each row: the same permissions, the server's answers in the dialog, and no new component. Invitations are a second `DataTable` inside a `Guard` for `users:invite`, since `perm` maps a resource name to `<name>:create`, not to `users:invite`.
- **Call (E2): `/account` shows the current session as "This one"**, with Sign out at the top through `signOut()`, so ending one's own session goes the way a Guard expects.
- **Call (E, the mail swap): dbtest's failing mail module is a never-started `*mail.Mailer`**, whose every `Send` is refused; the check that auth's `Stop` waits for register's and forgot's mailing is a unit test in `warp/auth`, since the real `Mailer` cannot be made to block.
- **Call (coordinator, after Track E): the log driver writes each mail to standard error as it is**, framed by `--- mail to <to>: <subject> ---` and `--- end of mail to <to> ---`, not through the log, so a link stays a line of its own whatever handler `middleware` gives the log (its text handler wrote the message's line breaks as `\n`, and a link copied from a terminal took them with it); the log keeps one `mail printed` line with `to` and `subject` and no body. `Printed()` and `mailtest` are unchanged.
- **Call (coordinator, final review M2): an API key cannot manage accounts or keys.** `invite`, `remove_invitation`, `update_user`, `remove_user`, `end_user_sessions`, `keys`, `create_key` and `remove_key` answer 403 `An API key cannot manage accounts or keys; sign in to do that.` to a request that carries a key, whatever its role, before the guard; a key may still read `users` and `invitations` as its role allows. A stolen administrator's key could otherwise invite its thief an account that outlives the key.
- **Call (E3): the shop writes `API_KEYS=on`**, in `scripts/shop.sh` and its tests, so its account page shows keys.

### The owner's decisions

Decided under the owner's standing order of 2026-11-28 ("work autonomous till v1.0"), by Fable; open to the owner's correction.

1. **`web/session.ts` and `nextPath` leave the template and the shop** once `@weft/ui` has them, removed in Tasks E1 and E3 (moved to the Trash in a worktree, never `rm`), with every import pointed at `@weft/ui`.
2. **`AUTH_REGISTRATION` defaults to `invite`; `open` stays an option.** Since every role reads every resource (`*:read`), the login guide says plainly, beside the setting, that `open` lets a stranger who registers read all of an app's records and suits only an app whose records every account may read. Registrants are `viewer`.
3. **A full Content-Security-Policy** (`script-src` with the build's inline-script hashes) comes with W7 in 0.6.0; 0.2.0 ships the policy in **Calls** without `script-src`.
4. **The per-account count stays**, as the roadmap asks; the login guide says that ten wrong tries in ten minutes keep an account from signing in with its password for that time, and that the reset link still signs the person in.
