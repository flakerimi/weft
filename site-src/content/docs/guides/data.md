---
title: Data
description: Records in SQLite through warp/data - a model, its table made at start, the five routes of a resource, lists with paging, sorting, search and filters, transactions and SQL of your own.
order: 24
---

On this page you learn how an app keeps its records: the `data` module opens the database and makes the tables, a store keeps one model in one table, and `data.Resource` gives it five routes. The package is `github.com/flakerimi/weft/warp/data`, on `database/sql` with no ORM. SQLite is the one engine today. The examples are from the shop in the repository's `examples/shop`.

## Setting it up

The driver is the app's dependency, not Warp's: `main.go` imports it, as `database/sql` intends.

```go
	// The SQLite driver, which warp/data opens by the name DB_DRIVER gives.
	_ "modernc.org/sqlite"
```

Then add it to `go.mod`, from the app's folder once `.weft/` is there (`weft create` and `weft unpack` make it):

```sh
go mod tidy
```

Without that step the build stops with `no required module provides package modernc.org/sqlite`.

`app/modules.go` lists `data.Module()` before every module that keeps records, and `.env` names the database:

```sh
DB_DRIVER=sqlite
DB_URL=file:data/shop.db
```

Registering the module does no work. The database is opened, its folder made and its tables brought up to date in the module's `Migrate`, before any module starts, so `weft gen` never makes a database. A missing value stops the app with one sentence, such as `warp: module "data" could not migrate: DB_DRIVER is not set; add DB_DRIVER=sqlite and DB_URL=file:data/app.db to .env.`; a driver `main.go` does not import is `no database driver is named "sqlite"; import one in main.go, such as _ "modernc.org/sqlite".`

## A model and its store

A model is a struct that embeds `data.Record` by value. The shop's product:

```go
// Product is one thing the shop sells. Its id and two times come from
// data.Record and are set by the store.
type Product struct {
	data.Record
	// Name is unique, sorted on and searched.
	Name string `json:"name" warp:"required,max=120,unique" data:"sort,search"`
	// Price is in the shop's currency, 0 or more, and sorted on.
	Price float64 `json:"price" warp:"min=0" data:"sort"`
	// Status is draft or live, and filtered on.
	Status string `json:"status" warp:"required,enum=draft|live" data:"filter"`
}
```

`data.Record` is `ID int64`, `CreatedAt` and `UpdatedAt`, answered as `id`, `created_at` and `updated_at`, all `readonly`: the store sets them, and a request never does. A module makes the store in its `Register` and keeps it:

```go
func (module) Register(app *warp.App) error {
	store := data.NewStore[Product](data.For(app), "products")
	data.Resource(app, "/products", store, data.PerAction(auth.Per("products")))
	return nil
}
```

The table's columns are the fields' `json` names. Text and times are `TEXT`, whole numbers and bools `INTEGER`, other numbers `REAL`, and `[]byte` `BLOB`. A pointer may be `NULL`; every other column is `NOT NULL` with the zero value as its default. A field tagged `json:"-"` is not kept, and a field of another type, such as a list or a struct, is a mistake that names it. A field with the rule `unique` gets a unique index, and a value another record has is answered 422 on that field, `This is already taken.`

At start a missing table is made and a missing column added. Nothing is ever dropped or retyped: a column whose type no longer matches stops the app with a sentence that names the table, the column and both types. Times are kept as text in UTC, to the nanosecond, so they sort as times.

## The five routes

`data.Resource(app, "/products", store, opts...)` registers, in its module, five routes named so:

| Name | Route | Takes | Answers |
| --- | --- | --- | --- |
| `list` | `GET /api/products` | a `data.Query` | 200, a `data.List[Product]` |
| `show` | `GET /api/products/{id}` | `data.ID` | 200, the record, or 404 |
| `create` | `POST /api/products` | the model, as its body | 201, the record made |
| `update` | `PATCH /api/products/{id}` | the id, and the model as its body | 200, the record with the sent fields written |
| `remove` | `DELETE /api/products/{id}` | `data.ID` | 204, or 404 |

`create` checks the model's rules and its `Validate` method, when it has one. `update` writes only the fields the body sent and checks only their rules; then, in one transaction, it lays them over the stored record, calls the model's `Validate` on that, and writes them. A missing id is `No product has id 7.`, a 404.

`data.Each(opts...)` gives all five routes the same route options, and `data.PerAction(fn)` each route the options `fn` returns for its action. `auth.Per("products")` is such a function: `products:read` for `list` and `show`, `products:create`, `products:edit` and `products:delete`.

## Lists

`list` takes `page` (from 1), `per_page` (20, at most 100), `sort`, `q` and `filter[...]`, and answers one page and how many records match in all:

```text
GET /api/products?q=lamp&sort=-price&filter[status]=live&per_page=1
```

```json
{"items":[{"id":3,"created_at":"2026-11-26T02:21:15.15703Z","updated_at":"2026-11-26T02:21:15.15703Z","name":"Desk lamp","price":75,"status":"live"}],"total":2,"page":1,"per_page":1}
```

A field takes part by its `data` tag: `sort`, `filter` or `search`, or several, comma-separated. `id`, `created_at` and `updated_at` can always be sorted on.

- `sort` is one field's `json` name, with `-` in front for descending; the default is `-id`, the newest first. Ties are in the order of their ids.
- `q` keeps the records in which any `search` field contains the text. Case is ignored for the ASCII letters alone, as SQLite's `LIKE` ignores it; `%` and `_` are taken literally.
- `filter[status]=live` keeps the records whose `filter` field equals the value, read as the field's kind; `filter[status]=` is no filter.

An empty table answers `{"items":[],"total":0,"page":1,"per_page":20}`. Anything else is one 400 that names each part, by its key in the query:

```json
{"error":{"code":"bad_request","message":"2 fields have a problem.","fields":{"filter[size]":"products cannot be filtered by \"size\"; use one of: status.","sort":"products cannot be sorted by \"cost\"; use one of: created_at, id, name, price, updated_at."}}}
```

A `sort` longer than 64 bytes, a filter value longer than 200 bytes, and a NUL byte in `q` or a filter value are 400s too. `page` and `per_page` are read as [query fields](routes.md#where-each-field-comes-from) and checked by `data.Query`'s rules, `min=1` and `max=100`, which pass a 0:

| Query | Answer |
| --- | --- |
| `page=0`, `page=`, or none | 200, page 1 |
| `page=-1` | 422, `page: Enter 1 or more.` |
| `page=two`, `page=1.5` | 400, `page: Send a whole number.` |
| a page past the last | 200, with no items and the total |
| `per_page=0`, `per_page=`, or none | 200, 20 a page |
| `per_page=-1` | 422, `per_page: Enter 1 or more.` |
| `per_page=101` | 422, `per_page: Enter 100 or less.` |
| `per_page=x` | 400, `per_page: Send a whole number.` |

A `q` longer than 200 characters is a 422 too.

## Your own routes

A store's methods are for routes the five do not cover, called with the handler's context:

| Method | Does |
| --- | --- |
| `List(ctx, q)` | A page, as `list` answers it. From Go a page below 1 is 1, and more than 100 a page is 100 |
| `Get(ctx, id)` | The record, or a `warp.NotFound` |
| `First(ctx, field, value)` | The record with the lowest id whose field, by its `json` name, equals the value, and whether there is one |
| `Count(ctx)` | How many records there are |
| `Create(ctx, &v)` | Adds `v` and sets its id and times |
| `Update(ctx, id, &v, fields)` | Writes the fields named, and `updated_at`, and returns the record as it is now |
| `Delete(ctx, id)` | Removes it, or a `warp.NotFound` |

`db.Tx(ctx, fn)` runs `fn` in a transaction, which it commits when `fn` returns nil and rolls back otherwise. Store calls made with `fn`'s context run in it, and a `Tx` inside a `Tx` joins the outer one. Only `Tx` ends the transaction: when the request ends, the statement running is interrupted and the transaction rolled back, but an `fn` that does not pass its context on holds the write lock until it returns. A failed commit is rolled back and the connection kept for the next request. A `PATCH` of your own lays the sent fields over the stored record and checks it there, since the core checks only what was sent ([Typed routes](routes.md#one-struct-for-create-and-update)). For a model `Good` with a `Validate` method that compares its price and its sale price:

```go
// PriceIn is the input of PATCH /api/goods/{id}/price.
type PriceIn struct {
	ID   int64 `path:"id"`
	Body Good  `body:"json"`
}

// price writes the sent fields, checking the record they make.
func (m *goods) price(ctx context.Context, in PriceIn) (Good, error) {
	var out Good
	err := m.db.Tx(ctx, func(ctx context.Context) error {
		g, err := m.store.Get(ctx, in.ID)
		if err != nil {
			return err
		}
		if warp.Sent(ctx, "price") {
			g.Price = in.Body.Price
		}
		if warp.Sent(ctx, "sale") {
			g.Sale = in.Body.Sale
		}
		if err := g.Validate(ctx); err != nil {
			return err
		}
		out, err = m.store.Update(ctx, in.ID, &g, warp.SentKeys(ctx))
		return err
	})
	return out, err
}
```

`m.db` is `data.For(app)` and `m.store` the store, both kept in `Register`. SQL of your own goes through `db.Exec`, `db.Query` and `db.QueryRow`, which run in the transaction on the context when there is one:

```go
_, err := m.db.Exec(ctx, `UPDATE "goods" SET "price" = "price" * 0.9 WHERE "status" = ?`, "live")
```

`db.SQL()` is the `*sql.DB` pool itself, which never runs in a transaction.

## SQLite's settings

Warp sets these on every connection, whatever `DB_URL` says: foreign keys on, the WAL journal, and transactions that take the write lock when they begin. It waits 5 seconds for a lock; a `DB_URL` may raise that, never lower it. Every spelling is read (`_busy_timeout`, `_timeout`, `_pragma=busy_timeout(N)`, escaped or repeated) and the largest of Warp's and the URL's is used. Each setting of the URL it replaces is logged at start, such as `data: DB_URL sets _busy_timeout=0, below Warp's 5000; 5000 is used, since the app's own writers wait on it.`, and a value it cannot read as `data: DB_URL sets _busy_timeout=x, which is not a busy timeout SQLite keeps; Warp's 5000 is used.`

A `_pragma` that names `journal_mode` or `locking_mode` stops the start: `warp: module "data" could not migrate: DB_URL sets journal_mode in _pragma=journal_mode(DELETE), and the store needs Warp's own; remove it from DB_URL.` Then the store reads its settings back from the database, and stops the start when one is still not what it needs, with a sentence that names it and `DB_URL`, such as `DB_URL sets busy_timeout to 0, and the store needs at least 5000; remove it from DB_URL.`

A call that still finds the database held by another connection after that wait is answered 503 `unavailable`, `The database is busy; try again in a moment.`, and logged at warn level, `data: the database was busy, so the request was answered 503`, with the method and the path.

## Testing

A test imports the driver too, and gives each test a database file of its own in `t.TempDir()`, as the shop's test in [Warp, the Go side](warp.md#testing-a-module) does. `db.SetClock(now)` replaces the clock the stores read the times from.

## Everything it exports

```go
package data // github.com/flakerimi/weft/warp/data

func Module() warp.Module
func For(app *warp.App) *DB
func (db *DB) Tx(ctx context.Context, fn func(ctx context.Context) error) error
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *Row
func (r *Row) Scan(dest ...any) error
func (r *Row) Err() error
func (db *DB) SQL() *sql.DB
func (db *DB) SetClock(now func() time.Time)

type Record struct {
	ID        int64     `json:"id" warp:"readonly"`
	CreatedAt time.Time `json:"created_at" warp:"readonly"`
	UpdatedAt time.Time `json:"updated_at" warp:"readonly"`
}
type ID struct{ ID int64 `path:"id"` }
type Query struct {
	Page    int               `query:"page" warp:"min=1"`
	PerPage int               `query:"per_page" warp:"min=1,max=100"`
	Sort    string            `query:"sort"`
	Q       string            `query:"q" warp:"max=200"`
	Filter  map[string]string `query:"filter"`
}
type List[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func NewStore[T any](db *DB, table string) *Store[T]
func (s *Store[T]) List(ctx context.Context, q Query) (List[T], error)
func (s *Store[T]) Get(ctx context.Context, id int64) (T, error)
func (s *Store[T]) First(ctx context.Context, field string, value any) (v T, ok bool, err error)
func (s *Store[T]) Count(ctx context.Context) (int, error)
func (s *Store[T]) Create(ctx context.Context, v *T) error
func (s *Store[T]) Update(ctx context.Context, id int64, v *T, fields []string) (T, error)
func (s *Store[T]) Delete(ctx context.Context, id int64) error

func Resource[T any](app *warp.App, path string, store *Store[T], opts ...ResourceOption)
func Each(opts ...warp.RouteOption) ResourceOption
func PerAction(fn func(action string) []warp.RouteOption) ResourceOption
```

The sections above say what each does; the [Go package reference](../reference/go.md) has `warp`'s.
