---
title: Typed routes
description: Routes as typed Go functions, their names and permissions, where each input field comes from, and updates of what was sent.
order: 23
---

On this page you learn how to write an API route in Go: a function that takes a context and an input struct and returns an output and an error, which Warp fills, checks and answers as JSON. The app and its modules are in [Warp, the Go side](warp.md).

## A route

Register a route with `warp.Get`, `warp.Post`, `warp.Put`, `warp.Patch` or `warp.Delete`, and write its path without `/api`, which the app adds:

```go
// Product is what the routes answer with.
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ShowIn is the input of GET /api/products/{id}.
type ShowIn struct {
	ID int `path:"id"`
}

// ListIn is the input of GET /api/products.
type ListIn struct {
	Page    int      `query:"page"`
	PerPage *int     `query:"per_page" warp:"min=1,max=100"`
	Tags    []string `query:"tag"`
}

// CreateIn is the body of POST /api/products.
type CreateIn struct {
	Name  string  `json:"name" warp:"required,max=120"`
	Price float64 `json:"price" warp:"min=0"`
	Plan  string  `json:"plan" warp:"enum=basic|pro"`
}

func (module) Register(app *warp.App) error {
	warp.Get(app, "/products", list, warp.Name("list"))
	warp.Get(app, "/products/{id}", show, warp.Name("show"))
	warp.Post(app, "/products", create, warp.Name("create"))
	warp.Delete(app, "/products/{id}", remove, warp.Name("remove"))
	return nil
}

func list(ctx context.Context, in ListIn) ([]Product, error) {
	return nil, nil // a nil slice is answered as []
}

func show(ctx context.Context, in ShowIn) (Product, error) {
	if in.ID != 1 {
		return Product{}, warp.NotFound(fmt.Sprintf("No product has id %d.", in.ID))
	}
	return Product{ID: 1, Name: "Lamp", Price: 12}, nil
}

func create(ctx context.Context, in CreateIn) (Product, error) {
	return Product{ID: 2, Name: in.Name, Price: in.Price}, nil
}

func remove(ctx context.Context, in ShowIn) (warp.None, error) {
	return warp.None{}, nil
}
```

`GET /api/products/1` answers `{"id":1,"name":"Lamp","price":12}`. The paths are the standard library's `http.ServeMux` patterns, wildcards included.

The output is written as JSON, with status 200. `warp.Post` answers 201 instead, and the option `warp.Status(200)` changes it. An output of `warp.None` answers 204 with no body; a route with an output and `Status(204)` or `Status(304)`, which carry none, answers 500 without running, and logs `warp: GET /api/x set the status 204 and has an answer to write; use warp.None as its output.` once. Every answer under `/api` carries `Cache-Control: no-store` unless the handler sets its own. An input of `warp.None` reads nothing from the request. An answer never holds `null` for a list, a map or bytes: a nil one, at any depth, is written as `[]`, `{}` or `""`, as the [generated types](types-from-go.md) say. A nil pointer is `null`, and a field tagged `omitempty` or `omitzero` is left out. A field typed `any` is written as it is.

### Names and permissions

Each route has a name within its module, which the [generated client](types-from-go.md) calls it by: `api.products.show(1)`. Without `warp.Name` it is made from the path's fixed segments, with the method in front unless it is `GET`: `GET /hello` is `hello`, `POST /hello/greet` is `post_hello_greet`. Two routes of one module with one name are a mistake, so `GET /products` and `GET /products/{id}`, both `products`, need a `warp.Name` each, as above; two given one name are told `give them different names.` Two routes that answer some of the same requests are `warp: route "GET /x" conflicts with GET /x, registered before it; give one of them another path.`

`warp.Permission("products:edit")` records the permission a route needs, for `app.Routes()` and `weft routes`, and guards nothing: `weft routes` shows it on a route that is open. The [login](login.md)'s `auth.Can("products:edit")` records it and guards the route. `warp.Options(opts...)` makes one option of several, which is how a package writes one like `Can`, and `warp.Requires("auth", "auth.Module()")` says the route needs a module: an app that lists none of that name has the mistake `warp: route "GET /x" needs the "auth" module; list auth.Module() in app/modules.go.`

### Where each field comes from

Every exported field of an input has exactly one of four tags, which says where it comes from:

| Tag | Filled from | Types |
| --- | --- | --- |
| `json:"name"` | The JSON body. Only `POST`, `PUT` and `PATCH` read one. | Anything `encoding/json` reads |
| `body:"json"` | The whole JSON body, on one field whose type is a struct (see below) | A struct |
| `query:"name"` | The query string | `string`, `bool`, any int, uint or float, a pointer to one of them, `[]string`, or `map[string]string` |
| `path:"name"` | The wildcard `{name}` of the path | `string`, `bool`, any int, uint or float |

The body is read first, then the query, then the path. The body is decoded into the `json` fields alone, so it cannot set a query or path field. A struct inside a body field is read from the body too: its fields need `json` tags, and take no `query` or `path` tag.

In the query, a pointer stays nil when the key is absent, and a `[]string` gets every value of a repeated key (`?tag=a&tag=b`). A key that is present but empty, such as `?page=`, counts as absent for a number or a bool. A `map[string]string` takes every `key[name]` of the query, key being its tag's name: `Filter map[string]string` tagged `query:"filter"` takes `?filter[status]=live` as `"status": "live"`. More than 32 names, or one longer than 64 bytes, is a 400.

### One struct for create and update

A field tagged `body:"json"` takes the whole body, so a route can take a record and an id from the path. Its fields are keyed as the input's own, `name` and not `body.name`, in errors and in `warp.Sent`. An input with a body field has no `json` fields of its own. A field with the rule `readonly`, at any depth, is answered and never read: a body that sends it, by its key or by a key that differs from it only in case, is a 400, and after decoding every readonly field is zero whatever was sent. `readonly` goes on a field, not on an embedded struct, and not inside a type that decodes itself (`UnmarshalJSON` or `UnmarshalText`), where Warp cannot see what it sets: both are registration mistakes.

```go
// Item is a record: its id is answered and never read.
type Item struct {
	ID    int64   `json:"id" warp:"readonly"`
	Name  string  `json:"name" warp:"required,max=120"`
	Price float64 `json:"price" warp:"min=0"`
	Sale  float64 `json:"sale" warp:"min=0"`
}

// UpdateIn is the input of PATCH /api/items/{id}: the id from the path, the item from the body.
type UpdateIn struct {
	ID   int64 `path:"id"`
	Body Item  `body:"json"`
}

func update(ctx context.Context, in UpdateIn) (Item, error) {
	item := Item{ID: in.ID, Name: "Lamp", Price: 30} // as stored
	if warp.Sent(ctx, "name") {
		item.Name = in.Body.Name
	}
	if warp.Sent(ctx, "price") {
		item.Price = in.Body.Price
	}
	if warp.Sent(ctx, "sale") {
		item.Sale = in.Body.Sale
	}
	// The record as it would be stored, checked whole.
	if err := item.Validate(ctx); err != nil {
		return Item{}, err
	}
	return item, nil
}
```

`POST` with `{"id":3,"name":"Lamp"}` to a route that takes an `Item` is a 400: `The body has the field "id", which this route does not take.`

On a route registered with `warp.Patch`, the rules run only on the `json` fields the body sent: `{"price":5}` leaves `name` alone, and `{"name":""}` still fails `required`. `warp.Sent(ctx, "price")` says whether the body had a key, and `warp.SentKeys(ctx)` lists them in the body's order. Query and path fields are checked as on every route. The input's own `Validate` method (below) runs, and sees zero for every `json` field the body left out, as the handler does; the body field's type's `Validate` does not run. So a `PATCH` route that checks fields against each other lays what was sent over the stored record and checks that, as `update` above does with `Item`'s `Validate` (below), and as [Data](data.md#your-own-routes)'s `data.Resource` does for you.

### JSON bodies only

A body that is not empty must be sent with `Content-Type: application/json`, or a type that ends in `+json`; anything else is answered 415. A page on another site can send a form or `text/plain` with the visitor's cookies and no preflight; taking JSON alone keeps it from sending a typed route a body. The app's [cross-site check](warp.md#cross-site-requests) covers a route that takes no body. An empty body reads as `{}`.

### When the input is wrong

The handler is not called, and the answer is in the [error shape](validation.md#errors):

| The request | Status, code | Message, fields |
| --- | --- | --- |
| A body larger than `MAX_BODY_MB` | 413 `too_large` | `The body is larger than 1 MB.` |
| A body that is not sent as JSON | 415 `unsupported_media_type` | `Send the body as application/json.` |
| A body nested more than 64 levels deep | 400 `bad_request` | `The body is nested more than 64 levels deep.` |
| A body that is not JSON, or has more after the value | 400 `bad_request` | `The body is not valid JSON.` |
| A body that is JSON but not an object | 400 `bad_request` | `The body must be a JSON object.` |
| A body field the input does not have, or a `readonly` one | 400 `bad_request` | `The body has the field "nme", which this route does not take.` |
| A body field of the wrong type, at any depth | 400 `bad_request` | `1 field has the wrong type.`, with `price: Send a number.` (or `Send text.`, `Send true or false.`, `Send a list.`, `Send an object.`, `Send a whole number from 0 to 255.`) |
| `null` for a field that is not a pointer, an `any` or a type with its own `UnmarshalJSON` other than `time.Time`, in a list or map too | 400 `bad_request` | the same, with `price: Send a value, not null.` |
| A list of another length for a fixed-size array | 400 `bad_request` | the same, with `rgb: Send a list of 3 values.` |
| A value its field's own decoder refuses (`"tomorrow"` for a `time.Time`) | 400 `bad_request` | the same, with `when: Send a time like 2026-11-25T10:00:00Z.` (or `Send base64 text.`, `Send a value this field takes.`); `The body has a value its field cannot take.` when no one field can be found |
| A query value that does not convert | 400 `bad_request` | `1 field has the wrong type.`, with `page: Send a whole number.` (or `Send a number.`, `Send true or false.`) |
| A path value that does not convert | 404 `not_found` | `Nothing is at /api/products/abc.` |
| A rule that fails | 422 `validation` | `1 field needs attention.`, with a message per field (see [Validation](validation.md#validation)) |

A field the body does not have is refused: a misspelt field dropped without a word is how an update loses data.

### The request and the answer's headers

A handler gets the request it is answering, and headers to send with its answer, from its context:

```go
func whoami(ctx context.Context, _ warp.None) (Product, error) {
	r := warp.Request(ctx)
	warp.ResponseHeader(ctx).Set("Cache-Control", "no-store")
	return Product{Name: r.Header.Get("User-Agent")}, nil
}
```

On `POST`, `PUT` and `PATCH` the request's body has already been read into the input, so there is nothing left in it to read. The headers are sent whether the handler succeeds or fails, so a `Set-Cookie` goes out with an error too.

### Mistakes in registering

A mistake in a route is found when it is registered, before any request: `warp.Get` and the rest return nothing, and `app.Err()` reports every mistake so far, one sentence each. `Serve` and `Start` return them all, so the app does not start. For example:

```text
warp: GET /products/{id}: BadIn.Name has no json, query, path or body tag; add one.
warp: GET /products/{id}: BadIn.Email is tagged json, and GET reads no body; tag it query or path.
warp: GET /products/{id}: BadIn.Email has the rule "emial", which warp does not know; use required, email, url, min=, max=, enum=, unique or readonly.
```

Two keys of one object that differ only in case, at any depth, are refused too, since `encoding/json` would take either for the other; `readonly` on an embedded struct or inside a type that decodes itself likewise:

```text
warp: POST /x: In has the fields Ab and AB, whose keys differ only in case; rename one.
warp: POST /x: zSelf decodes itself, so Warp cannot see what it sets, and zSelf.ID is tagged readonly; drop the tag or the type's own decoding.
warp: POST /x: In embeds FoldBase tagged readonly; tag the fields inside it, or give the field a name.
```

Besides those, a registration is refused when a field has two of the tags, a `json` tag has no name or a name that is not letters, digits, `_`, `-` and `.`, two fields have the same key, a `path` tag names no wildcard of the path or a wildcard has no field, a query or path field has a type it cannot be filled into, an input has two body fields or a body field and `json` fields, a struct inside a body field has a field without a `json` tag or with a `query`, `path` or `body` tag, the input embeds a pointer to a struct, or a struct with a `json` name or `json:"-"`, a rule does not fit its field, `warp.Status` is given a code outside 200 to 599, a path starts with `/api/`, or the same route is registered twice.

## Validation and errors

The rules a field's `warp` tag can hold, the messages they answer with, checks across fields, and the one error shape every failure is answered in, are in [Validation and errors](validation.md).
