# Warp: one slice, end to end

Date: 2026-11-25
Extends `2026-11-23-warp-design.md`. It replaces that design's order of phases W2 to W5 with one thin slice through all of them, and adds what the slice needs that no phase named. The layers are widened afterwards, each in its own plan.

## Intent

W1 delivered the core: an app, modules, typed routes, validation, one error shape. Its reviews found nearly every real defect where two pieces met. The remaining phases were planned as layers, each finished before the next, so nothing would have worked end to end until W5 and every meeting of pieces would have come late.

The slice is one small back-office app that lives in the repository and that the test suite builds and drives: one resource, stored in a database, behind a login, with its TypeScript types and form rules generated from the Go types, a page with a table and an edit dialog, and a page for one record. Every part is as small as it can be while still being real. It exists to find what the layers get wrong about each other before any of them is widened, and to be the working code that `weft scaffold` is later written from.

The owner chose this order on 2026-11-25 ("slice first, build it"). No first real app was named, so the slice's app is a generic one, a shop with products.

## Goals

- `examples/shop` runs: `weft dev` in it shows a login, a products table with create, edit and delete, and a page per product, with no fetch call or HTTP handler written by hand for the resource.
- A renamed Go field is a TypeScript error in the page after the next `weft gen`.
- The suite builds and drives the app on every run, so a later change that breaks a seam fails a test.
- Every piece of the core that later phases need from W1's notes is settled here, on real use: route names and permissions, updates that touch only what was sent, how a module gets the database, cross-site protection.

## Non-goals (each is a later widening)

- PostgreSQL and MySQL; relations (`belongs_to`, `has_many`, `many_to_many`); soft delete.
- Registration, invitations, password reset, roles other than the owner, the `mail` module.
- `weft scaffold`, and lifting the table and dialog into `DataTable` and `EntityDialog`. The slice writes them plainly in the app with the components `@weft/ui` has (`Table`, `Form`, `FormField`, `Input`, `InputNumber`, `Dialog`).
- A client-side router and layouts. Navigation is plain links between pages.
- Attachments, events, live data, jobs, translated fields, OpenAPI.
- Changing the app template `weft create` writes. It stays the `hello` app until the slice's shapes have settled.
- A seeding interface, an audit log, CSV export, webhooks.

## Anti-goals

- A slice that only works because the example app knows the framework's insides. The app uses what a developer would use.
- A second way to do something W1 already does (a second validation path, a second error shape, a router beside `ServeMux`).
- Anything in `warp/` that imports a module outside the standard library.

## Constraints

- Every rule in `AGENTS.md` and the Warp design's Rules 1 to 7. In particular: `warp/` is standard library only; a database driver is the app's dependency; errors are one sentence that names the thing and says what to do; nothing registers itself from `init`.
- Go 1.26, so `net/http.CrossOriginProtection`, `crypto/pbkdf2` and `os.Root` are available.
- The W1 limits stand: a 1 MiB body, 64 levels of nesting, at most 100 listed fields.
- What `weft` generates is regenerated and never edited by hand; `app/` and `web/` belong to the developer and are never overwritten. That one rule is the whole stability promise before 1.0.

## Approaches

1. **The phases as planned** (W2, then W3, W4, W5). Clean units, but nothing end to end until W5, and W2 would fix the generated types' shape before any table or form had used them.
2. **One thin slice, then widen.** Chosen by the owner, and recommended by both agents who reviewed W1. Every seam is met early on something that runs; the cost is a first pass that is rough on purpose and partly rewritten when widened.
3. **Let the first real app pull features.** Nothing unused gets built, but one app's deadlines would shape the framework.

## The design

### 1. What the core gains (`warp/`)

**Route names, modules and permissions.** `Route` gains `Module` (the name of the module whose `Register` was running when the route was added; empty for a route added outside one), `Name` and `Permission`. Two exported options set them: `warp.Name("list")` and `warp.Permission("products:edit")`. A route without `warp.Name` is named from its method and path: the static path segments joined with `_`, with the method in front unless it is `GET` (`GET /hello` is `hello`; `POST /hello/greet` is `post_hello_greet`). Two routes of one module with the same name are a registration mistake. `warp.Permission` only records; the module that enforces it (auth) supplies the option that also guards.

**What a module depends on.** Two lifetimes, two mechanisms, and nothing else:

- For the app's lifetime (the database, later the mailer): `warp.Provide(app, value)` stores one value per Go type, and `warp.Need[T](app)` returns it or records one mistake naming the module that asked and the module to list first. A module wraps these in its own typed function, so app code reads `data.For(app)`. There is no lookup by string and no package-level state.
- For one request (the signed-in user, the open transaction): values on the `context.Context` the handler receives, behind a function of the owning module (`auth.User(ctx)`).

**Updates that touch only what was sent.** The core records which top-level keys a JSON body had: `warp.Sent(ctx, "name") bool` and `warp.SentKeys(ctx) []string`. On a route registered with `warp.Patch`, rules run only on the fields the body sent, so one struct serves create and update: omitting `name` leaves it alone, and sending `"name": ""` still fails `required`. `Validate(v)` called directly checks everything, as today. The difference lives in `Patch` and is stated in its doc comment.

**Checks across fields.** An input type may have a method `Validate(ctx context.Context) error`. It runs after the tag rules pass. Returning `warp.Invalid(fields)` answers 422 with those fields; any other error is answered as any handler error is.

**`filter[...]` in the query.** A `query` field may be a `map[string]string`: `Filter map[string]string \`query:"filter"\`` takes `filter[status]=live&filter[category]=3`. Keys are bounded like everything else from a request (at most 32 entries; a longer key than 64 bytes is a 400).

**Cross-site requests.** Every request under `/api` with a method other than `GET`, `HEAD` or `OPTIONS` goes through `http.CrossOriginProtection` before routing. One from another site is answered 403, code `forbidden`, `This request came from another site.` `TRUSTED_ORIGINS` in `.env` (comma-separated origins) lets named sites through, for an app whose pages are served elsewhere. This closes the gap W1's review found: a route that takes no body was open.

**The app describes itself.** `(*App).Describe() Description` returns every route (module, name, method, path, status, permission, input type, output type) and every struct type they reach, each field with its JSON, query or path name, its TypeScript type, whether it may be absent, and its parsed rules. It is a file of the core, not a package of its own: the core must print it on request and cannot import a package that imports it. When the environment has `WARP_DESCRIBE=1`, `Serve` writes the description as JSON to standard output and returns without listening.

### 2. Data (`warp/data`)

- **Opening.** `data.Module()` reads `DB_DRIVER` and `DB_URL` from `.env`, opens the database with `database/sql`, and provides a `*data.DB`. The app's `main.go` imports the driver, as `database/sql` intends. A missing driver or value is one sentence at start.
- **A store.** `data.NewStore[T](db, "products")` for a struct with an `ID int64`. Columns are the fields' `json` names. `CreatedAt` and `UpdatedAt` of type `time.Time`, when present, are set by the store. Methods: `List(ctx, Query)`, `Get(ctx, id)`, `Create(ctx, *T)`, `Update(ctx, id, *T, fields []string)` (only the named fields are written), `Delete(ctx, id)`. A missing record is `warp.NotFound`.
- **Tables.** At start a missing table is created and a missing column is added. A column whose type no longer matches stops the app with one sentence naming the table, the column and both types. Nothing is ever dropped or retyped.
- **Lists.** `data.Query` is an ordinary typed input: `Page`, `PerPage` (default 20, at most 100), `Sort` (`name`, `-created_at`), `Q`, `Filter`. A field takes part only if its `data` tag says so: `data:"sort,filter,search"`. Sorting or filtering on any other field is a 400 that names it. The answer is `data.List[T]{Items, Total, Page, PerPage}`.
- **Transactions.** `db.Tx(ctx, func(ctx context.Context) error)` commits when the function returns nil and rolls back otherwise. A store call made with that context uses the transaction.
- **Already taken.** A field with the rule `unique` gets a unique index. A violation is answered 422 on that field, `This is already taken.`, not 500.
- **The five routes.** `data.Resource(app, "/products", store, opts...)` registers `list`, `show`, `create`, `update` (a `PATCH` writing the sent fields) and `remove`, named so. It lives in `data`, not in the core, because the core cannot import `data`. An option gives per-action route options, which is how a permission per action is attached.
- **Engine.** SQLite only, behind a small dialect type so the next engines add a dialect and do not touch the store. Its tests need a driver, which `warp/` may not import, so they live in a nested Go module, `dbtest/`, with its own `go.mod` (the pure-Go SQLite driver, and a `replace` to the repository). The suite runs it beside `go test ./...`.

### 3. Login (`warp/auth`)

The smallest login that is honest about security:

- **Users** in a table through `data`: email (unique), a password hash, a role.
- **Passwords** are hashed with PBKDF2-HMAC-SHA256 from the standard library, 600,000 iterations, a random salt, stored with a prefix that names the scheme and its parameters so a later scheme can sit beside it.
- **Sessions** are a signed cookie (HMAC-SHA256 with `AUTH_SECRET`): the user's id and an expiry. `HttpOnly`, `SameSite=Lax`, `Secure` when the request was HTTPS, lasting `AUTH_SESSION_DAYS` (default 30). `AUTH_SECRET` shorter than 32 bytes stops the app at start.
- **Routes** under `/api/auth`: `login`, `logout`, `me`. A wrong email and a wrong password get the same answer. After 5 failed tries for one email and address within a minute, further tries get 429 for the rest of that minute.
- **The first user.** `AUTH_OWNER_EMAIL` names the owner. If no user exists at start, the app logs a one-time setup link; opening it sets the owner's password. Until then login is impossible, which is the point.
- **Guarding.** `auth.Can("products:edit")` is a route option: it records the permission on the route and refuses a request with no session (401) or without the permission (403). In the slice the owner has every permission and there is no other role. `auth.User(ctx)` gives the signed-in user. `auth.Per("products")` gives `data.Resource` a permission per action: `products:read`, `products:create`, `products:edit`, `products:delete`.
- **Vue side**, in the example app for now: a `useSession()` that asks `/api/auth/me`, a login page and a setup page.

### 4. Types from Go (`weft gen`, `weft routes`)

- `weft gen` builds the app with the `dev` tag, runs it with `WARP_DESCRIBE=1`, and writes `web/.weft-types/api.ts`: an interface per Go type; `api.<module>.<name>(...)`, one typed function per route, which sends JSON and throws an `ApiError` carrying the status, code, message and fields of the error shape; and `schema`, the rules of each input, for a form to check in the browser and to show the server's answer on the same field.
- `weft dev` runs it at start and after each Go change that builds; `weft build` runs it before the page build. The file is not committed.
- `weft routes` prints each route with its method, path, module, name and permission.
- The generator is Go, in `cmd/weft`, and writes TypeScript as text.

### 5. A page for a record made after the build

A page whose path has a parameter (`pages/products/[id].vue`) must today list its paths at build. A record made after the build has no page.

A dynamic page that exports `runtime = true` in place of `getPaths()` is built once, as a shell, and `routes.json` lists it with its pattern. `warp.Serve` answers a path with that shell when no exact page matches. In the browser, `useParams()` reads the parameters from the address, using the pattern the build wrote into the page. `weft dev` serves such a page the same way. A dynamic page with neither export still fails the build with today's sentence.

The shell is answered with 200 whatever the id; a record that does not exist is the API's 404, which the page shows. The server answering 404 for the document would mean asking the app on every page request, which the serving design forbids. (Claude's call; recorded for the owner.)

### 6. The app (`examples/shop`)

A Go module of its own (its `go.mod` holds the SQLite driver and a `replace` to the repository), with `web/` a member of the pnpm workspace as `site-src` is. Modules listed in `app/modules.go`: `data`, `auth`, `products`. A product has a name (required, unique, at most 120 characters), a price (0 or more), a status (`draft` or `live`) and its two timestamps.

Pages: `/login`; `/setup`; `/products`, a table with paging, sorting, search and a status filter, and a dialog to create and edit that checks the generated schema and shows the server's field errors; `/products/[id]`, a runtime page showing one product.

Tests, run by the suite: the whole story over HTTP in Go (set up the owner, sign in, create, list with sort, search and filter, update one field leaving the others, a duplicate name answering 422 on `name`, delete, the same calls refused without a session, a request from another site refused); the generated `api.ts` type-checked with the pages that use it; and the app built and its pages requested.

### 7. Testing other apps (`warp/warptest`)

`warptest.New(t, modules...)` gives an app and a client that keeps cookies, with helpers that send JSON and decode the answer or the error shape. The database is passed in, since the driver is the app's. `auth` adds `authtest.Owner(t, client)`, which makes an owner and signs in. The example app's tests are written with them.

## Decided by the owner

- The order: one slice, then widen (2026-11-25).
- Build it without waiting for approval of this document (2026-11-25, "build it").

## Calls made by Claude (open to correction)

- The slice's app is a shop with products, since no first app was named.
- The app lives in `examples/shop` with its own `go.mod`; store tests live in `dbtest/`. Both because `warp/` may not depend on a driver.
- `Describe` is part of the core, not `warp/describe`.
- `data.Resource`, not `warp.Resource`.
- A route's default name comes from its method and path.
- On `PATCH`, rules run only on sent fields; elsewhere nothing changes.
- Cross-site protection is on by default for every state-changing API request.
- PBKDF2 at 600,000 iterations; a signed cookie holding the user id and expiry, with no server-side session table, so signing out everywhere is not possible in the slice.
- The owner is set up through a one-time link in the log.
- A record page's shell is answered 200 for any id.
- The app template is not changed by the slice.

## Agreed with Argon (the second agent, in `coord.md`, 2026-11-25)

Dependencies by lifetime; transactions and a test kit inside the slice; cross-field validation and unique violations as field errors; `map[string]string` for `filter[...]`; module, name and permission on `Route`; `http.CrossOriginProtection`; PBKDF2 from the standard library; record pages as pattern-matched shells with an explicit opt-in; no stability machinery before 1.0.

## Left to the builder

- Exact SQL, and the column type for each Go type in SQLite.
- The JSON layout of the description, beyond what section 1 says it must contain.
- The text layout of the generated `api.ts`.
- How `useParams()` learns the page's pattern.
- The wording of sentences this document does not quote, under the one-sentence rule.

## After the slice

Each of these is its own plan, in the order the first real app needs them: more engines and relations; registration, reset and roles with `mail`; `weft scaffold` with `DataTable` and `EntityDialog`, written from this app; the template gaining what has settled; then W6 and W7 as designed.
