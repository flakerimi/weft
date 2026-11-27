---
group: data
purpose: A CRUD resource table with search, filters, EntityDialog and ConfirmDialog.
use: For a resource table backed by a generated Warp client (`api.<plural>`) and schema (`schema.<Name>Input`), with server-paged listing, search, filters, creation, editing and deletion.
avoid: For a read-only table over an array or a custom fetcher without CRUD dialogs, use Table; for a few key metrics, use StatCard.
adapts:
  input: Inherits Table's, Button's and the dialog controls' touch hit areas (at least 44 px, with row action buttons switching to `md` on coarse pointers).
  surface: Nothing changes in behaviour; `size` sets the row height and text follows the surface's type scale.
  width: When its own box is narrow (under 640 px), Table switches rows to cards, toolbar filter definitions with four or fewer choices switch from a horizontal RadioGroup to a Select, and EntityDialog goes full-screen.
---
It has no stylesheet of its own: it composes `Stack`, `FormField`, `Input`,
`RadioGroup`, `Select`, `Button`, `Table`, `EmptyState`, `EntityDialog` and
`ConfirmDialog`.

`DataTable` pairs a toolbar, `Table`, `EntityDialog` and `ConfirmDialog`
around a generated Warp resource (`{ list, create, update, remove }`) and its
field schema (`schema.ProductInput` from `weft gen`). Pass
`:resource="api.products"` (or individual `fetch`, `create`, `update` and
`remove` functions) and `:schema="schema.ProductInput"`.

**Toolbar.** When `searchable` is true (the default), the toolbar renders a
search `Input` bound to `v-model:search` with `search-debounce` (default
250 ms). Each entry in `filter-defs` (`{ key, label, items, allLabel, control }`)
renders a filter control bound into `v-model:filters`, prepending
`allLabel || labels.anyFilter` (default `"Any"`, value `""`) unless
`allLabel: false`; filters with four or fewer options render a horizontal
`RadioGroup` on regular and wide containers and a `Select` when narrow. When
`create` and `schema` (or `#dialog`) are present, a `"New <name>"` button
opens `EntityDialog` in create mode.

**Columns and row actions.** `columns` takes the same typed array as `Table`,
and every `#<key>-cell` and `#<key>-header` slot forwards to `Table`.
Unslotted cells format numbers in the `WeftProvider`'s locale (`en-US` by
default) and booleans with `labels.booleanYes` / `labels.booleanNo`. When
`update` (with `schema` or `#dialog`) or `remove` is present and `columns` has
no `actions` column, `DataTable` appends a trailing `Actions` column with
`Edit` (`aria-label` `"Edit <itemLabel>"`) and `Delete` (`aria-label`
`"Delete <itemLabel>"`).

**By permission.** `perm` takes a resource name, such as `perm="products"`:
`"New <name>"` is drawn only for a signed-in user with `products:create`,
`Edit` only with `products:edit` and `Delete` only with `products:delete`
(`useSession().can`, always true for the role `owner`), and the `Actions`
column is dropped when neither row action is left. It hides; it does not
disable, and the server still decides: a route guarded by `auth.Per` answers
403 to anyone without the permission. Pages are built with no session, so the
server draws none of the three, and neither does the browser's first render;
they appear once the table has mounted and `GET /api/auth/me` has answered,
so hydration always matches. Without `perm` nothing is asked.

**Dialogs and refresh.** Saving in `EntityDialog` emits `saved` and refreshes
`Table`. Confirming in `ConfirmDialog` calls `remove(row[idKey])`, emits
`deleted`, refreshes the table, and moves focus to the next row's action
button (or `"New <name>"` when empty); if `remove` rejects, the dialog stays
open with an `Alert` (`role="alert"`). `defineExpose` provides `refresh()`,
`edit(record)` and `askRemove(record)`.
