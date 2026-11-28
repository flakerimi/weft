---
title: weft scaffold
description: Every field type and modifier of weft scaffold, the files it writes, and what it refuses.
order: 36
---

Writes one resource into an app `weft create` made, then runs `weft gen`. The other commands are in [CLI](cli.md).

```text
weft scaffold <name> <field[:type[:modifier...]]>... [--plural <word>] [--api]
weft scaffold product name:string:required:max=120:unique price:float:min=0 sale:float:nullable status:enum=draft,live:required
```

`<name>` is singular snake_case (`line_item`). Its plural, made by a fixed rule (`person` → `people`, `category` → `categories`, `box` → `boxes`, else `s`) or given with `--plural`, names the package `app/<plural>`, the table, `/api/<plural>` and the permissions. Refused: `data`, `auth`, `warp`, `app`, `web`, `test`, `main`, Go's words and standard packages, names the story test uses, and names whose type Weft or the client has (`user`, `module`, `record`, `list`, `login`…).

A field's type is written, never guessed from its name; with none it is `string`.

| Type | Go | Rule | `data` tag |
| --- | --- | --- | --- |
| `string` | `string` | | `sort,search` |
| `text` | `string` | | `search` |
| `email`, `url` | `string` | `email`, `url` | `sort,search` |
| `int`, `float` | `int64`, `float64` | | `sort` |
| `bool` | `bool` | | `sort,filter` |
| `date`, `datetime` | `time.Time`, RFC 3339 | `date` on a `date` | `sort` |
| `enum=a,b` | `string` | `enum=a\|b` | `sort,filter` |

The modifiers, in any order after the type, are `nullable`, a pointer that may be `null` (`sale:float:nullable` is `*float64`); `required` (not on `bool` or with `nullable`); `unique` (not on `bool`, `text`, `date`, `datetime`); and `min=N` and `max=N`, in characters for text (not on `bool`, `date`, `datetime`, `enum`); a field has only the rules written. Enum choices are separated by commas: `|` or `(` is refused (`weft scaffold: write enum choices separated by commas after enum=, as in status:enum=draft,live.`), and `?` with `:nullable`.

| It writes | What it is |
| --- | --- |
| `app/<plural>/model.go` | The model, embedding `data.Record` |
| `app/<plural>/routes.go` | `data.Resource`, with `data.PerAction(auth.Per("<plural>"))` when `app/modules.go` lists `auth.Module()`, else open, which it says |
| `app/<plural>/<plural>_test.go` | The story over HTTP on SQLite: make two, list sorted, searched and filtered, change, delete, and, with `auth`, 401 without a session |
| `app/modules.go` | The import and `<plural>.Module()` last in the list; the rest, comments too, kept |
| `web/components/<Plural>Table.vue`, `<Singular>View.vue` | Islands: a `DataTable` over `api.<plural>` and `schema.<Singular>Input`, and one record; with `auth`, each in a `Guard` for `<plural>:read`, the table with `perm="<plural>"` |
| `web/pages/<plural>/index.vue`, `[id].vue` | The table's page and the record's run-time page |

`--api` writes only the Go. A `date` or `datetime` not `required` is a pointer, so an emptied control clears it, and a table with one calls `useDateFields()`. It lists what it wrote under `Scaffolded <name> as <plural>:`.

Nothing is written (one sentence, exit 1) when `app/<plural>` is there, or without `--api` the page folder, `web/pages/<plural>.vue` or an island, or when `app/modules.go` has no `func Modules() []warp.Module` returning a `[]warp.Module{...}` with `data.Module()`. `id`, `created_at` and `updated_at` (from `data.Record`), `actions` (`DataTable`'s), a field given twice, an unknown type or modifier, or a modifier its type does not take exit 2. A later release's type exits 1 naming it, as `weft scaffold: field "maker" has type "belongs_to", and relations land in Weft 0.3.0 (W3); use a scalar type for now.`: `belongs_to`, `has_many`, `many_to_many` (0.3.0), `image`, `file` (0.4.0), `json`, `translated` (0.6.0).
