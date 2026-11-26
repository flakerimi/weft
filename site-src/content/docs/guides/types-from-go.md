---
title: Types from Go
description: weft gen writes the TypeScript types, a typed function per route and the form rules from the app's Go types, so a renamed Go field is a type error in the page.
order: 26
---

On this page you learn how the Vue side of an app calls its Go API with no fetch written by hand: `weft gen` asks the app to describe its routes and types and writes `web/.weft-types/api.ts`, which the pages import. A renamed Go field is then a TypeScript error in every page that reads it.

## Where the file comes from

`weft gen` builds the app with the `dev` tag and runs it with `WARP_DESCRIBE=1`, so `Serve` prints every route and every type it takes and answers as JSON, and exits without opening anything. From that it writes `web/.weft-types/api.ts`. `weft dev` writes it again after each Go change that builds, and `weft build` before the pages, which it then type-checks against it, so you seldom run `weft gen` yourself. An app that serves without a `warp.App` gets a client with no routes. The file is git-ignored; never edit it. The [CLI reference](../reference/cli.md#weft-gen) has what each command prints and its errors.

`weft routes` prints the same description as a table. The shop's:

```text
METHOD  PATH                MODULE    NAME    PERMISSION
POST    /api/auth/login     auth      login   -
POST    /api/auth/logout    auth      logout  -
GET     /api/auth/me        auth      me      -
POST    /api/auth/setup     auth      setup   -
GET     /api/products       products  list    products:read
GET     /api/products/{id}  products  show    products:read
POST    /api/products       products  create  products:create
PATCH   /api/products/{id}  products  update  products:edit
DELETE  /api/products/{id}  products  remove  products:delete
```

## What it writes

An interface for each Go type a route takes or answers, under its Go name, with its JSON names. A type with `readonly` fields also gets an `…Input` twin without them, which is what a route takes. From the shop's [`Product`](data.md#a-model-and-its-store):

```ts
/** shop/app/products.Product */
export interface Product {
  id: number
  created_at: string
  updated_at: string
  name: string
  price: number
  status: string
}

/** shop/app/products.Product as a route takes it: without its readonly fields. */
export interface ProductInput {
  name: string
  price: number
  status: string
}
```

Then `api`, a function for each typed route, under its module's name and the route's ([Names and permissions](routes.md#names-and-permissions)); routes outside any module are under `api.app`:

```ts
    /** GET /api/products/{id} (products:read) */
    async show(id: number): Promise<Product> {
      return send('GET', `/api/products/${segment('id', id)}`)
    },
    /** PATCH /api/products/{id} (products:edit) */
    async update(id: number, body: Partial<ProductInput>): Promise<Product> {
      return send('PATCH', `/api/products/${segment('id', id)}`, { body })
    },
```

Path parameters come first, in the path's order, then the body, then the query, which may be left out: `api.products.list({ page: 2, sort: '-price', filter: { status: 'live' } })`. A `PATCH` takes a `Partial` body. A route that answers `warp.None` resolves to `undefined`.

Text and times are `string`, numbers `number`, a pointer `T | null`, a list `T[]` and a map `Record<string, T>`. A pointer field may be left out, `name?:`, and so may an answered field tagged `omitempty` or `omitzero`, and a field an answered type has from an embedded pointer, which a nil pointer leaves out. A list, a map or bytes is never `null` on the wire, since [the server never answers one so](routes.md#a-route), except in a field typed `any`.

- A 64-bit integer is `number`, which is exact only up to 2^53: an id above 9007199254740991 is read as another.
- A fixed-size array, `[3]int`, is `number[]`; the server takes a list of exactly 3.
- `null` is sent only for a pointer, an `any` or a type with its own `UnmarshalJSON` other than `time.Time`; for anything else the server refuses it on the field with `Send a value, not null.`
- A type whose `MarshalJSON` or `MarshalText` has a pointer receiver is written by it, since an answer is written through a pointer; held as a map's values, where nothing can point to it, it is a mistake `weft gen` names.
- A generic type is named with its arguments: `List[Product]` is `ListProduct`, and `Page[*Address]` is `PageAddressOrNull`.

## Calling the API

```ts
import { api, ApiError } from '../.weft-types/api.ts'

try {
  const product = await api.products.show(id)
} catch (e) {
  if (e instanceof ApiError && e.status === 404) missing.value = true
}
```

Every rejection is an `ApiError`, with `status`, `code`, `message` and `fields` from the [error shape](routes.md#errors). When nothing could be sent, the status is 0 and the code `network`. What the server would misread is refused before anything is sent, with status 0 and code `bad_request`: a path parameter that is empty, `.` or `..`, which would reach another route (`The id in the address cannot be empty, "." or "..".`), or is not well-formed text; a path or query parameter that is `undefined`, `null` or not a finite number (`The id in the address is NaN; give it a value.`); and a body with `NaN`, an infinity or a `bigint` (`fields: { price: 'Send a number.' }`) or an `undefined` in a list, which JSON would send as `null`. A failure that is not in the error shape has the code `http` and the status's text.

Requests go to the page's own origin, with its cookies. `setBase(url)` puts a path prefix, or an absolute URL of that same origin, in front of every path; an API on another origin is not supported yet.

## Tables and forms

`Table` asks its `fetch` for a page in the list route's own names, `{ page, per_page, sort?, q?, filter? }`, and reads `{ items, total }`, so the generated function fits as it is. From the shop's products table, where `ready` turns true once the session is known to be signed in:

```vue
    <Table
      v-if="ready"
      ref="table"
      :fetch="api.products.list"
      :columns="columns"
      :search="search"
      :filters="filters"
      :per-page="10"
      default-sort="name"
      caption="Products"
      hide-caption
    >
```

`schema` holds the rules of each body type, by type and field name, which `Form` checks before it sends: `<Form :state="draft" :schema="schema.ProductInput" @submit="save">`. What the server refuses comes back as an `ApiError` whose `fields` the form shows on the same fields, through its ref, `form = useTemplateRef<FormApi>('form')`:

```ts
try {
  await api.products.create(values)
} catch (e) {
  if (e instanceof ApiError) form.value?.setErrors(e.fields)
}
```

`unique`, and every rule of a type nested in a body, are checked by the server only, and arrive the same way, as `This is already taken.` under the name field. A `Form` sends only from its island, so put it in one ([Islands](pages-and-islands.md#islands)); in development, a submit of one in no island says so in the console.

### Where the browser and Go differ

`Form` and Go check the same rules, with a few differences. The server's answer is the one that counts, and it lands on the field:

| Value | `Form` in the browser | Go |
| --- | --- | --- |
| `0` for a `required` number | filled in, so it passes | empty, so it fails (`This is required.`) unless the field is a pointer |
| `0` for a number with `min=1` | checked, so it fails | empty, so it passes unless the field is a pointer |
| `""`, `false` or `[]` for a `required` pointer field | empty, so it fails | sent, since a pointer that is not nil says sent, so it passes |
| `""` for a pointer field with `min`, `email` or `enum` | empty, so it passes | sent, so it is checked and fails |
| an empty object, or a zero time, for `required` | filled in | empty, so it fails |
| `pattern` | checked | no such rule |

`email` and `url` are one rule on both sides, tested on the same rows. A model that accepts 0 does not mark the field `required`; one that must tell 0 from nothing uses a pointer.

## A renamed field

Rename `json:"price"` to `json:"cost"` in the shop's `Product`, save, and `weft dev` writes the file again; `pnpm typecheck` then fails in each `.vue` file that reads `price`, and so does `weft build`, before it builds anything: `weft build: the pages do not type-check; fix what vue-tsc says above and run weft build again.` A `Table` column's `key` and a `FormField`'s `name` are plain strings, which no type check catches. The full `go test ./cmd/weft` does the same to a copy of the shop, with `weft gen` and the type check (`-short` skips it).
