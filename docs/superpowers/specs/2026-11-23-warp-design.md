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
| `auth` | `AUTH=on`, `AUTH_SECRET`, `AUTH_SESSION_DAYS`, `AUTH_REGISTRATION=open\|invite\|closed` | Register, login, logout, current user, forgot and reset password, change password, profile and avatar. Sessions in a signed HttpOnly cookie. Roles Owner, Administrator, Member, Viewer, each including the next one's permissions; permissions named `resource:action`; the first user is the Owner. `auth.Can("products:edit")` on a route. Vue: `useSession()`, a login page, `perm` on `Button` and row actions, a route guard. |
| `storage` | `STORAGE=local`, `STORAGE_DIR`, `STORAGE_MAX_MB` | Attachments on a record's `image` and `file` fields, with type and size limits per kind, served under `/api/files/`. Vue: `FileUpload` bound to the field. |
| `mail` | `MAIL=smtp\|log`, `MAIL_HOST`, `MAIL_PORT`, `MAIL_USER`, `MAIL_PASSWORD`, `MAIL_FROM` | Send text and HTML email from templates. `log` prints the message, for development. Password reset uses it. |
| `events` | none | `events.On("products.created", fn)` and `Emit`. `data.Store` emits `<resource>.created`, `.updated`, `.deleted`. |
| `live` | `LIVE=on` | Server-sent events at `/api/live`, by topic, checked against the session's permissions. An event can be published to a topic in one line, so a table refreshes when a record changes. Vue: `useLive`, `useLiveSeries`. |
| `jobs` | `JOBS=on` | Work on a schedule (`every 5m`, `daily 03:00`), with a list of jobs and their last result. |
| `i18n` | `LOCALES=en,sq` | Translated fields on a record. |
| `middleware` | `LOG=json\|text`, `CORS_ORIGINS`, `RATE_LIMIT`, `API_KEYS=on` | Request logging on `log/slog`, security headers, CORS, a rate limit that works, API keys for other programs. |
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
| W4 | A first pass delivered by the slice: `warp/auth` with one owner, a signed-cookie session, `auth.Can` and `auth.Per`. Left: roles, registration, reset, `mail`, `useSession`, `perm`, a route guard in Vue. As designed: `auth`: sessions, users, roles, permissions; `useSession`, the login page, `perm`, the route guard; `mail` with the `log` and `smtp` drivers |
| W5 | Not started; the slice's shop is the code it is to be written from. As designed: `weft scaffold`, with `DataTable` and `EntityDialog` (from the Weft design's phase 5) |
| W6 | `storage` and attachments; `events` and `live` with `useLive` |
| W7 | `jobs`, `i18n`, `middleware` (rate limit, API keys), `docs` |

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
