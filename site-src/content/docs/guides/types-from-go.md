---
title: Types from Go
description: weft gen writes the TypeScript types, a typed function per route and the form rules from the app's Go types, so a renamed Go field is a type error in the page.
order: 26
---

On this page you learn how the Vue side of an app calls its Go API with no fetch written by hand: `weft gen` asks the app to describe its routes and types and writes `web/.weft-types/api.ts`, which the pages import. A renamed Go field is then a TypeScript error in every page that reads it.

## Where the file comes from

`weft gen` builds the app with the `dev` tag and runs it with `WARP_DESCRIBE=1`, so `Serve` prints every route and every type it takes and answers as JSON, and exits without opening anything. From that it writes `web/.weft-types/api.ts`. `weft dev` writes it again after each Go change that builds, and `weft build` before the pages, so you seldom run `weft gen` yourself. The file is git-ignored; never edit it. The [CLI reference](../reference/cli.md#weft-gen) has what each command prints and its errors.

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

Text and times are `string`, numbers `number`, a pointer `T | null`, a list `T[]` and a map `Record<string, T>`. A pointer field may be left out, `name?:`, and so may an answered field tagged `omitempty`. A list, a map or bytes is never `null` on the wire, since [the server never answers one so](routes.md#a-route), except in a field typed `any`.

## Calling the API

```ts
import { api, ApiError } from '../.weft-types/api.ts'

try {
  const product = await api.products.show(id)
} catch (e) {
  if (e instanceof ApiError && e.status === 404) missing.value = true
}
```

Every rejection is an `ApiError`, with `status`, `code`, `message` and `fields` from the [error shape](routes.md#errors). When nothing could be sent, the status is 0 and the code `network`. A path parameter that is empty, `.` or `..` would reach another route, so it is refused before anything is sent, with code `bad_request`: `The id in the address cannot be empty, "." or "..".` A failure that is not in the error shape has the code `http` and the status's text.

Requests go to the page's own origin, with its cookies. `setBase(url)` puts a path prefix, or an absolute URL of that same origin, in front of every path; an API on another origin is not supported yet.

## Tables and forms

`Table` asks its `fetch` for a page in the list route's own names, `{ page, per_page, sort?, q?, filter? }`, and reads `{ items, total }`, so the generated function fits as it is. From the shop's products table:

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

`unique`, and every rule of a type nested in a body, are checked by the server only, and arrive the same way, as `This is already taken.` under the name field. A `Form` sends only from its island, so put it in one ([Islands](pages-and-islands.md#islands)).

### Where the browser and Go differ

`Form` and Go check the same rules, with a few differences. The server's answer is the one that counts, and it lands on the field:

| Value | `Form` in the browser | Go |
| --- | --- | --- |
| `0` for a `required` number | filled in, so it passes | empty, so it fails (`This is required.`) unless the field is a pointer |
| `0` for a number with `min=1` | checked, so it fails | empty, so it passes unless the field is a pointer |
| `"a b"@example.com` for `email` | refused (the rule browsers use for `type="email"`) | taken (`net/mail`) |
| `a..b@example.com` for `email` | taken | refused |
| `pattern` | checked | no such rule |

A model that accepts 0 does not mark the field `required`; one that must tell 0 from nothing uses a pointer.

## A renamed field

Rename `json:"price"` to `json:"cost"` in the shop's `Product`, save, and `weft dev` writes the file again; `pnpm typecheck` then fails in each `.vue` file that reads `price`. The suite does exactly that to a copy of the shop on every run.
