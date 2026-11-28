---
title: Validation and errors
description: The rules a typed route's input is checked by, checks across fields, and the one error shape every failure is answered in.
order: 24
---

On this page you check a [typed route](routes.md)'s input and answer its failures: the rules, a check across fields, and the error shape.

## Validation

Rules go in a field's `warp` tag, separated by commas. They run on every typed input after it is filled and before the handler:

| Rule | For | Fails when | Message |
| --- | --- | --- | --- |
| `required` | anything | the value is zero, an empty slice or map, a nil pointer, or text that is empty once trimmed | `This is required.` |
| `email` | text | it is not an email address by the rule browsers use for `type="email"`, which `Form` uses too: ASCII, no display name, nothing trimmed | `Enter an email address.` |
| `url` | text | it is not an `http://` or `https://` address with a host, as the browser's `Form` defines it | `Enter a web address that starts with http:// or https://.` |
| `min=N` | text, a number, a list | it has fewer than N characters, is below N, or has fewer than N items | `Use at least N characters.`, `Enter N or more.`, `Choose at least N.` |
| `max=N` | the same | more than N | `Use at most N characters.`, `Enter N or less.`, `Choose at most N.` |
| `enum=a\|b` | text, a number | it is not one of them | `Choose one of: a, b.` |
| `date` | a `time.Time` | it is not midnight in UTC, written in UTC | `Send a date alone, such as 2026-03-29T00:00:00Z.` |
| `unique` | text, a number | another record has the value: checked by [Data](data.md)'s store, nothing else | `This is already taken.` |
| `readonly` | anything | never: the field is answered and not read ([one struct for create and update](routes.md#one-struct-for-create-and-update)) | |

- Every rule but `required` passes an empty value, so an optional field may be left out: an empty `email` passes, and a 0 passes `min=1`.
- A pointer that is not nil was sent: `required` passes, and the other rules check what it points at, so a `*int` holding 0 fails `min=1`. A pointer field is how an input tells "sent as 0" from "not sent".
- Characters are Unicode code points, not bytes: `çikë` is 4.
- Only the first rule a field fails is reported, in the order of the tag.
- A field's key is its `json`, `query` or `path` name. A field of a struct inside is `address.city`, of a list of structs `lines.0.sku`, and of a map's value `by_name.x.sku`.
- At most 100 fields are listed, with 16 KiB of keys in all, counted before JSON escaping, so the answer stays under about 100 KB, and none over 1 KiB, in field, index and sorted key order. When one is left out the message is `More fields need attention than are listed; the first N are listed.`, or with none listed `More fields need attention than can be listed.`

For the body `{"name":"","price":-1,"plan":"gold"}`, `POST /api/products` answers 422 with:

```json
{"error":{"code":"validation","message":"3 fields need attention.","fields":{"name":"This is required.","plan":"Choose one of: basic, pro.","price":"Enter 0 or more."}}}
```

A check across fields is a method `Validate(ctx context.Context) error` on the input, or on its body field's type. It runs once the tag rules pass; `warp.Invalid` answers 422 with its fields, and any other error is answered as the handler's would be:

```go
// Validate runs once the tag rules pass, on the Item of Typed routes.
func (i Item) Validate(ctx context.Context) error {
	if i.Sale > i.Price {
		return warp.Invalid(map[string]string{"sale": "Make the sale no more than the price."})
	}
	return nil
}
```

`warp.Validate(v)` checks any struct against its tags the same way, every field, and returns nil or the 422 error.

## Errors

Every error the API answers has one shape, with `Content-Type: application/json`:

```json
{"error":{"code":"not_found","message":"No product has id 7."}}
```

`fields` is there only when it has something, as in the 422 above. A handler chooses its answer by returning one of these:

| Function | Status | Code |
| --- | --- | --- |
| `warp.BadRequest(message)` | 400 | `bad_request` |
| `warp.Unauthorized(message)` | 401 | `unauthorized` |
| `warp.Forbidden(message)` | 403 | `forbidden` |
| `warp.NotFound(message)` | 404 | `not_found` |
| `warp.Conflict(message)` | 409 | `conflict` |
| `warp.TooLarge(message)` | 413 | `too_large` |
| `warp.UnsupportedMedia(message)` | 415 | `unsupported_media_type` |
| `warp.Invalid(fields)` | 422 | `validation`, with the message `1 field needs attention.` or `N fields need attention.` |
| `warp.TooMany(message)` | 429 | `rate_limited` |
| `warp.Unavailable(message)` | 503 | `unavailable`, for a moment's trouble, such as a busy database |

Each returns a `*warp.Error`, which is found even when it is wrapped, as by `fmt.Errorf("load: %w", err)`. Any other error, and a panic, is logged with the method and the path, and answered 500, code `internal`, with `Something went wrong on the server; its log has the details.` and none of the error's text. The next request is served as usual.

The app answers some errors itself:

- a path under `/api` that no route has: 404 `not_found`, `Nothing is at /api/nope.`;
- a method a route does not take: 405 `method_not_allowed`, `/api/products/1 does not take PUT; use DELETE or GET.`, with an `Allow` header;
- a state-changing request from another site: 403 `forbidden`, `This request came from another site.`;
- a path that is not clean, such as `/api/a//b`, or a `{rest...}` route asked without its tail: the mux's redirect, with its status and `Location`, and code `moved`, `This is at /api/a/b; ask there.`

An `*Error` with a status alone answers that status's words, as code `teapot` and `I'm a teapot.` for 418.

`warp.WriteError(w, r, err)` writes an error in this shape from a plain handler, and nothing for `context.Canceled` once the client has gone; `warp.WriteJSON(w, status, v)` writes a value as JSON, a nil slice at the top, or one `v` points to, as `[]`; inside `v`, nil lists and maps are `null` as `encoding/json` writes them. An `*Error` with a status outside 400 to 599, or a value JSON cannot hold, such as a NaN, is answered as the 500.
