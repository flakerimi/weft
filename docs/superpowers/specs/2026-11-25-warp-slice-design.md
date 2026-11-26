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

**Route names, modules and permissions.** `Route` gains `Module` (the name of the module whose `Register` was running when the route was added; empty for a route added outside one), `Name` and `Permission`. Two exported options set them: `warp.Name("list")` and `warp.Permission("products:edit")`. A route without `warp.Name` is named from its method and path: the static path segments joined with `_`, with the method in front unless it is `GET` (`GET /hello` is `hello`; `POST /hello/greet` is `post_hello_greet`). Two routes of one module with the same name are a registration mistake. `warp.Permission` only records; the module that enforces it (auth) supplies the option that also guards, built with `warp.Options(opts...)`, which makes one route option out of several. Without it a package outside the core could not write an option that both records and guards.

**What a module depends on.** Two lifetimes, two mechanisms, and nothing else:

- For the app's lifetime (the database, later the mailer): `warp.Provide(app, value)` stores one value per Go type, and `warp.Need[T](app, "data")` returns it, or returns the zero value and records one mistake: `module "products" needs the "data" module; list data.Module() before it in app/modules.go.` The caller names the module it needs, because a module that is not listed yet cannot name itself. A module wraps these in its own typed function, so app code reads `data.For(app)`. There is no lookup by string and no package-level state.
- For one request (the signed-in user, the open transaction): values on the `context.Context` the handler receives, behind a function of the owning module (`auth.User(ctx)`).

**Updates that touch only what was sent.** The core records which top-level keys a JSON body had: `warp.Sent(ctx, "name") bool` and `warp.SentKeys(ctx) []string`. On a route registered with `warp.Patch`, rules run only on the fields the body sent, so one struct serves create and update: omitting `name` leaves it alone, and sending `"name": ""` still fails `required`. `Validate(v)` called directly checks everything, as today. The difference lives in `Patch` and is stated in its doc comment.

**Checks across fields.** An input type may have a method `Validate(ctx context.Context) error`. It runs after the tag rules pass. Returning `warp.Invalid(fields)` answers 422 with those fields; any other error is answered as any handler error is.

**`filter[...]` in the query.** A `query` field may be a `map[string]string`: `Filter map[string]string \`query:"filter"\`` takes `filter[status]=live&filter[category]=3`. Keys are bounded like everything else from a request (at most 32 entries; a longer key than 64 bytes is a 400).

**Cross-site requests.** Every request under `/api` with a method other than `GET`, `HEAD` or `OPTIONS` goes through `http.CrossOriginProtection` before routing. One from another site is answered 403, code `forbidden`, `This request came from another site.` `TRUSTED_ORIGINS` in `.env` (comma-separated origins) lets named sites through, for an app whose pages are served elsewhere. This closes the gap W1's review found: a route that takes no body was open.

**The app describes itself.** `(*App).Describe() Description` returns every route (module, name, method, path, status, permission, input type, output type) and every struct type they reach, each field with its JSON, query or path name, its TypeScript type, whether it may be absent, and its parsed rules. It is a file of the core, not a package of its own: the core must print it on request and cannot import a package that imports it. When the environment has `WARP_DESCRIBE=1`, `(*App).Serve` writes the description as JSON to standard output and returns nil without migrating, starting or listening. It does so after its check of `Err()`, so a registration mistake still fails `weft gen` with its sentence.

**One struct for create and update.** Three small general things, so the core never learns what a record is:

- *A body field.* An input may have one field tagged `body:"json"`. It receives the whole JSON object, and its fields are keyed without a prefix in errors and in `Sent` (`name`, not `body.name`). An input with a body field has no `json` fields of its own. A struct may hold a field of a type parameter where it may not embed one, so a generic package can write `struct { ID int64 \`path:"id"\`; Body T \`body:"json"\` }`. The tag leaves room for `body:"raw"` later.
- *A `readonly` rule.* A field with `warp:"readonly"` is answered but never read: it is left out of the body a route takes (sending it is the usual 400, "does not take"), and out of the generated input type and schema.
- *`data.Record`* (section 2), which a model embeds by value.

**Behind a proxy.** `TRUST_PROXY=on` in `.env` makes the app believe `X-Forwarded-Proto` and the last address of `X-Forwarded-For`; it is off by default, since a client can send both. `warp.ClientIP(r)` and `warp.IsHTTPS(r)` give the answer either way. `APP_URL` is the address people open the app at (default `http://localhost:` and the port); `weft dev` sets it to the page server's address.

**Starting without listening.** `(*App).Start(ctx)` runs the modules' `Migrate` and `Start`, and `(*App).Stop(ctx)` their `Stop`, as `Run` does around serving. A test that uses `Handler()` calls them; `Run` is built on them.

### 2. Data (`warp/data`)

- **Opening.** `data.Module()` reads `DB_DRIVER` and `DB_URL` from `.env` and provides a `*data.DB`. Registering does no file or network work: the database is opened, and tables are made, in the module's `Migrate`, which runs before any module starts. So `weft gen` never creates a database. The app's `main.go` imports the driver, as `database/sql` intends. A missing driver or value is one sentence at start.
- **A record.** A model embeds `data.Record` by value: `ID int64`, `CreatedAt` and `UpdatedAt`, with the JSON names `id`, `created_at` and `updated_at`, all `readonly`. `data.ID` is the input `struct { ID int64 \`path:"id\"\` }`.
- **A store.** `data.NewStore[T](db, "products")` for a struct that embeds `data.Record`. It records the table it needs on the `*data.DB`; nothing is opened. Columns are the fields' `json` names. The two timestamps are set by the store. Methods: `List(ctx, Query)`, `Get(ctx, id)`, `Create(ctx, *T)`, `Update(ctx, id, *T, fields []string)` (only the named fields are written), `Delete(ctx, id)`. A missing record is `warp.NotFound`.
- **Tables.** At start a missing table is created and a missing column is added. A column whose type no longer matches stops the app with one sentence naming the table, the column and both types. Nothing is ever dropped or retyped.
- **Lists.** `data.Query` is an ordinary typed input: `Page`, `PerPage` (default 20, at most 100), `Sort` (`name`, `-created_at`), `Q`, `Filter`. A field takes part only if its `data` tag says so: `data:"sort,filter,search"`. Sorting or filtering on any other field is a 400 that names it. The answer is `data.List[T]{Items, Total, Page, PerPage}`.
- **Transactions.** `db.Tx(ctx, func(ctx context.Context) error)` commits when the function returns nil and rolls back otherwise. A store call made with that context uses the transaction.
- **Already taken.** A field with the rule `unique` gets a unique index. A violation is answered 422 on that field, `This is already taken.`, not 500.
- **The five routes.** `data.Resource(app, "/products", store, opts...)` registers `list`, `show`, `create`, `update` (a `PATCH` writing the sent fields) and `remove`, named so. `create` takes the model as its body; `update` takes the id from the path and the model as its body, writing only the sent fields; `show` and `remove` take `data.ID`. It lives in `data`, not in the core, because the core cannot import `data`. An option gives per-action route options, which is how a permission per action is attached.
- **Engine.** SQLite only, behind a small dialect type so the next engines add a dialect and do not touch the store. Its tests need a driver, which `warp/` may not import, so they live in a nested Go module, `dbtest/`, with its own `go.mod` (the pure-Go SQLite driver, and a `replace` to the repository). The suite runs it beside `go test ./...`.

### 3. Login (`warp/auth`)

The smallest login that is honest about security:

- **Users** in a table through `data`: email (unique), a password hash, a role.
- **Passwords** are hashed with PBKDF2-HMAC-SHA256 from the standard library, 600,000 iterations, a random salt, stored with a prefix that names the scheme and its parameters so a later scheme can sit beside it.
- **Sessions** are a signed cookie (HMAC-SHA256 with `AUTH_SECRET`): the user's id and an expiry. `HttpOnly`, `SameSite=Lax`, `Secure` when the request was HTTPS, lasting `AUTH_SESSION_DAYS` (default 30). `AUTH_SECRET` shorter than 32 bytes stops the app at start.
- **Routes** under `/api/auth`: `login`, `logout`, `me`, and `setup`, which takes the one-time token and a password. A wrong email and a wrong password get the same answer. After 5 failed tries for one email and address within a minute, further tries get 429 for the rest of that minute. The address is `warp.ClientIP(r)`, without its port.
- **The first user.** `AUTH_OWNER_EMAIL` names the owner. If no user exists at start, the app logs a one-time setup link, `APP_URL` and `/setup#token=…` (in the fragment, so it reaches no proxy log or `Referer`; this was `?token=` until the login's review); the page there sets the owner's password. The token is random, kept only in memory, made afresh at each start while no user exists, and useless once the owner exists. Until then login is impossible, which is the point.
- **Guarding.** `auth.Can("products:edit")` is a route option: it records the permission on the route and refuses a request with no session (401) or without the permission (403). In the slice the owner has every permission and there is no other role. `auth.User(ctx)` gives the signed-in user, read from the users table on each guarded request, so a removed user is out at once. `auth.Per("products")` gives `data.Resource` a permission per action: `products:read`, `products:create`, `products:edit`, `products:delete`.
- **Vue side**, in the example app for now: a `useSession()` that asks `/api/auth/me`, a login page and a setup page.

### 4. Types from Go (`weft gen`, `weft routes`)

- `weft gen` builds the app with the `dev` tag, runs it with `WARP_DESCRIBE=1`, and writes `web/.weft-types/api.ts`: an interface per Go type; `api.<module>.<name>(...)`, one typed function per route, which sends JSON and throws an `ApiError` carrying the status, code, message and fields of the error shape; and `schema`, the rules of each input, for a form to check in the browser and to show the server's answer on the same field.
- `weft dev` runs it at start and after each Go change that builds; `weft build` runs it before the page build. The file is not committed.
- `weft routes` prints each route with its method, path, module, name and permission.
- The generator is Go, in `cmd/weft`, and writes TypeScript as text.

### 5. A page for a record made after the build

A page whose path has a parameter (`pages/products/[id].vue`) must today list its paths at build. A record made after the build has no page.

A dynamic page that exports `runtime = true` in place of `getPaths()` is built once, as a shell, with empty parameters, into a folder that keeps the bracket name (`pages/products/[id]/index.html`, since a colon is not a safe file name), and `routes.json` lists it with its pattern and `runtime: true`. The build writes the pattern onto the page's root element, which is how `useParams()` knows it. `warp.Serve` answers a path with that shell when no exact page matches. In the browser, `useParams()` reads the parameters from the address, using the pattern the build wrote into the page. `weft dev` serves such a page the same way. A dynamic page with neither export still fails the build with today's sentence.

The shell is answered with 200 whatever the id; a record that does not exist is the API's 404, which the page shows. The server answering 404 for the document would mean asking the app on every page request, which the serving design forbids. (Claude's call; recorded for the owner.)

### 6. The app (`examples/shop`)

It is shaped exactly as `weft create` shapes an app, because `weft dev`, `weft gen` and `weft build` must run in it and `weft scaffold` is later written from it: a `package.json` with its `"weft"` settings beside `main.go`, a `go.mod` that holds the SQLite driver and replaces the Weft module with `./.weft/go`, and `.weft/` git-ignored. It is not a member of the repository's pnpm workspace. The suite drives it as it drives a created app: a test stages a bundle, copies `examples/shop` to a temporary folder, and runs `weft build`, the app's Go tests, the type check and the built binary there. `scripts/shop.sh` does the same staging for a person and runs `weft dev` in the folder itself. Modules listed in `app/modules.go`: `data`, `auth`, `products`. A product has a name (required, unique, at most 120 characters), a price (0 or more), a status (`draft` or `live`) and its two timestamps.

Pages: `/login`; `/setup`; `/products`, a table with paging, sorting, search and a status filter, and a dialog to create and edit that checks the generated schema and shows the server's field errors; `/products/[id]`, a runtime page showing one product.

Tests, run by the suite: the whole story over HTTP in Go (set up the owner, sign in, create, list with sort, search and filter, update one field leaving the others, a duplicate name answering 422 on `name`, delete, the same calls refused without a session, a request from another site refused); the generated `api.ts` type-checked with the pages that use it; and the app built and its pages requested.

### 7. Testing other apps (`warp/warptest`)

`warptest.New(t, modules...)` gives an app and a client that keeps cookies, with helpers that send JSON and decode the answer or the error shape. The database is passed in, since the driver is the app's. `auth` adds `authtest.Owner(t, client)`, which makes an owner and signs in. The example app's tests are written with them.

### 8. What `@weft/ui` changes

The slice may change `@weft/ui` where it disagrees with Warp's wire format, since they are one system:

- `Table`'s server-side fetch sends `per_page`, as the Warp design's list query has it, where it sent `perPage`; its docs stop naming `weft.Resource`.
- `Form`'s field schema gains `enum` and `url`, so every rule the generator writes can be checked in the browser.
- One difference stays and is written down: for `required`, `Form` counts a typed `0` as filled in, and Go counts a number that is not a pointer and is 0 as empty. The server's answer then appears on the field. A model that accepts 0 does not mark the field `required`.

## Decided by the owner

- The order: one slice, then widen (2026-11-25).
- Build it without waiting for approval of this document (2026-11-25, "build it").

## Calls made by Claude (open to correction)

- The slice's app is a shop with products, since no first app was named.
- The app lives in `examples/shop` with its own `go.mod`, shaped as a created app; store tests live in `dbtest/`. Both because `warp/` may not depend on a driver. `AGENTS.md` gains the two nested modules, their one allowed dependency, and their place in the test list.
- One struct for create and update through a body field, a `readonly` rule and `data.Record`, not through a special case in the core.
- `TRUST_PROXY` is off by default; `APP_URL` names where the app is opened.
- `@weft/ui` changes where it disagrees with the wire format.
- `Describe` is part of the core, not `warp/describe`.
- `data.Resource`, not `warp.Resource`.
- A route's default name comes from its method and path.
- On `PATCH`, rules run only on sent fields; elsewhere nothing changes.
- Cross-site protection is on by default for every state-changing API request.
- PBKDF2 at 600,000 iterations; a signed cookie holding the user id and expiry, with no server-side session table, so signing out everywhere is not possible in the slice.
- The owner is set up through a one-time link in the log.
- A record page's shell is answered 200 for any id.
- The app template is not changed by the slice.

## Calls made while building (2026-11-25, open to correction)

Each was made by the coordinator or an implementer and held by a review; the task reports in `.superpowers/sdd/2026-11-25-warp-slice/` have the detail.

- **Names.** Default route names clash only within a module; routes outside one are not checked. `GET /products` and `GET /products/{id}` both default to `products`, so a module with both names one of them.
- **Mistakes from outside the core.** `(*App).Mistake(format, args...)` records a registration mistake for a package such as `data`, with `warp: ` in front.
- **The description** carries what the generator needs beyond section 1: `typed` (false for `Handle`), each parameter's `style` and rules, each type's Go name, and an `…Input` twin for a body type that has readonly fields at any depth. Two types of one name, a half encoding, a map keyed by anything but text or a whole number, and `,string` are mistakes.
- **Answers never hold `null`** for a list, a map or bytes, at any depth, so the generated types are true on the wire; a field typed `any`, and `WriteJSON` called directly, are the limits.
- **Readonly fields are zeroed after decoding**, whatever the decoder chose; the body scan only decides the 400. A key that folds onto a readonly field's is refused at every level. The review showed one build flag (`GOEXPERIMENT=jsonv2`) makes the scan and the decoder disagree.
- **On `PATCH` the body type's `Validate` does not run**, since the body holds zero for what was left out; `data.Resource`'s update lays the sent fields over the stored record and validates that, in the transaction that writes it.
- **The `url` rule** is one written definition, the same in Go and in `Form`.
- **The data layer.** Warp's SQLite settings (`_foreign_keys`, WAL, `_txlock=immediate`) win over a `DB_URL`'s, and a `DB_URL` may raise the 5 s busy timeout, never lower it (the owner's ruling reversed after a probe showed the store's own writers depend on it). A database held past it answers 503 `unavailable` (`warp.Unavailable`), logged at warn. `DB.Exec`, `Query` and `QueryRow` join the transaction on the context; `SQL()` is the pool. `Store.First` and `Store.Count` were added for the login.
- **The login.** The cookie holds the user's id, the nanosecond the user was made and an expiry, so a replaced database refuses old cookies; under HTTPS it is `__Host-session`. Tries are counted before any lookup or hash: 5 a minute per email and address, 30 a minute per address, the address being `ClientKey` (an IPv6 client's /64). At most `GOMAXPROCS` hashes run at once; a login that finds no slot within 3 s is a 503 and costs no try. `LoginInput` and `SetupInput` are exported so the generated client names them; `auth.AddUser`, `auth.RemoveUser` and `auth.SetupToken` are for tools and tests.
- **The generated client.** `schema` is typed by an annotation rather than `as const`, so it is assignable to `FieldSchema`. Every `api` function is async and every rejection an `ApiError` (`network` when nothing could be sent, `bad_request` for an empty or dot path parameter, before sending). `setBase` takes a path prefix or a URL of the pages' own origin; an API on another origin is not supported. Names TypeScript cannot use are refused with a sentence.
- **`@weft/ui`.** `Table` hands `fetch` the wire's names, `{ page, per_page, sort?, q?, filter? }`. `Form` is `method="post"` behind an inline pre-paint script that cancels a Weft form's submit until its island is live, rather than disabled, so typing and password managers work; the island runtime keeps what was typed before hydration. A `Form` therefore needs an island.
- **Run-time pages.** Of two patterns that match one path, the one with more fixed segments wins, then one without a catch-all, then fewer parameters, except that of two catch-alls the one with more wins, so a catch-all never hides a more specific page.
- **`weft dev`** sets `APP_URL` to the page server's address unless its own environment has a non-empty one; a development `.weft/VERSION` carries the bundle's hash, so a changed bundle is unpacked again; `weft unpack` unpacks without installing; the app has 5 s to describe itself after each build.
- **The shop.** `pnpm-lock.yaml` is git-ignored there, since its packages come from `.weft/`; `status` is `required`.

## What the slice taught

From the shop's list (Task 12), with what became of each in the consolidation wave (`wave-D-report.md`).

| # | What the app had to do | Now |
| --- | --- | --- |
| 1 | Translate `Table`'s query names to the list route's | Fixed: `TableQuery` has the wire's names; `:fetch="api.products.list"` |
| 2 | An `enum` field is `string` in TypeScript, not its values | Open |
| 3 | A generated input type cannot be a form's state (`InputNumber` gives `null`) | Open: the shop's `Draft` type and `price ?? 0` |
| 4 | Work out by hand which fields changed, for `PATCH` | Open: `changes()` in `ProductsTable` |
| 5 | `useParams()` gives strings where the client takes numbers | Partly: the shop checks the id (`productId`), the client refuses empty and dot parameters; a typed `useParams` is open |
| 6 | A body field with no control has nowhere to show its error | Open: `SetupForm` moves `fields.token` to the top |
| 7 | Before hydration a `Form` was a GET form that put a password in the address | Fixed: `method="post"` and the pre-paint guard; typed values survive hydration |
| 8 | Every form writes the same catch for an `ApiError` | Open: the shop's `formProblem` |
| 9 | A `Form`'s template ref had no type | Fixed: `FormApi` |
| 10 | A dialog's form is remounted to forget its errors | Open: `formKey` |
| 11 | The table's search asks on every keystroke | Open |
| 12 | Every app with a login writes the same session code | Open: `web/session.ts` and `nextPath` in `web/answers.ts` |
| 13 | A guarded island waits for the session before it asks | Open: the `ready` flag |
| 14 | Islands share module state only because the build made one chunk | Open: a doc line and a test |
| 15 | `weft dev` did not set `APP_URL` | Fixed |
| 16 | A development `.weft/` was never unpacked again | Fixed: the bundle's hash |
| 17 | Nothing only unpacked `.weft/` | Fixed: `weft unpack` |
| 18 | A driver's `go.sum` cannot be written before `.weft/go` exists | Open |
| 19 | Nothing writes the secrets an app needs | Open: `scripts/shop.sh` and the test write `.env` |
| 20 | No layouts: the header is repeated | Open, a non-goal |
| 21 | `pnpm-lock.yaml` would go stale with `.weft/` | Open, the owner's call |
| 22 | Money is a `float64` | Open |
| 23 | The test kit takes raw, hand-escaped paths | Open |
| 24 | What worked in one line: the model and its guarded routes in 12 lines of Go, the story test, `unique` on the form's field, `runtime = true`, a renamed field breaking the type check | Not a defect |
| 25 | An open redirect in the shop's `nextPath` | Fixed in the shop |
| 26 | A network failure showed fetch's own words | Fixed in the shop |
| 27 | The setup token stayed in the address | Fixed in the shop (`history.replaceState`) |
| 28 | `next` nested deeper on each press of Sign in | Fixed in the shop |
| 29 | The shop's `AGENTS.md` did not say to unpack `.weft/` before `go test` | Fixed |

The workarounds still standing in the shop, each something the framework still makes an app do by hand: the `Draft` type and `price ?? 0` (3), `changes()` (4), `formKey` (10), the `ready` flag (13), moving `fields.token` to the form's top (6), the session code and `nextPath` (12), and `scripts/shop.sh test` running `weft gen` so the web tests and the type check have the client.

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
