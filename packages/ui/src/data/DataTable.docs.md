---
group: data
purpose: A CRUD resource table with search, filters, EntityDialog and ConfirmDialog.
use: For a resource table backed by a generated Warp client (`api.<plural>`) and schema (`schema.<Name>Input`), with server-paged listing, search, filters, creation, editing and deletion.
avoid: For a read-only table over an array or a custom fetcher without CRUD dialogs, use Table; for a few key metrics, use StatCard.
adapts:
  input: Inherits Table's, Button's and the dialog controls' touch hit areas (at least 44 px).
  surface: Nothing changes in behaviour; `size` sets the row height and text follows the surface's type scale.
  width: When its own box is narrow (under 640 px), Table switches rows to cards, toolbar filter definitions with four or fewer choices switch from a horizontal RadioGroup to a Select, and EntityDialog goes full-screen.
---
It has no stylesheet of its own: it composes `Stack`, `FormField`, `Input`,
`RadioGroup`, `Select`, `Button`, `Table`, `EmptyState`, `EntityDialog` and
`ConfirmDialog`.

`DataTable` composes a toolbar, `Table`, `EntityDialog` and `ConfirmDialog`
around a generated Warp resource (`{ list, create, update, remove }`) and its
field schema (`schema.ProductInput` from `weft gen`). Pass
`:resource="api.products"` (or individual `fetch`, `create`, `update` and
`remove` functions) and `:schema="schema.ProductInput"`.

**Toolbar.** When `searchable` is true (the default), the toolbar renders a
search `Input` bound to `v-model:search` and forwarded to `Table` with
`search-debounce` (default 250 ms). Each entry in `filter-defs`
(`{ key, label, items, control }`) renders a filter control bound into
`v-model:filters`, prepending `labels.anyFilter` (default `"Any"`, value
`""`) when no item has an empty value; filters with four or fewer options
render a horizontal `RadioGroup` on regular and wide containers and a
`Select` when narrow. When `create` (or `resource.create`) is present, a
`"New <name>"` button at the end of the toolbar opens `EntityDialog` in
create mode. The `toolbar` slot adds controls beside search and filters, and
`toolbar-actions` adds buttons beside `"New <name>"`.

**Columns and row actions.** `columns` takes the same typed array as `Table`,
and every `#<key>-cell` and `#<key>-header` slot forwards directly to
`Table`. Unslotted cells format numbers with `toLocaleString()` and booleans
with `labels.booleanYes` / `labels.booleanNo` (`"Yes"` / `"No"`). When
`update` or `remove` is present and `columns` has no `actions` column,
`DataTable` appends a trailing `Actions` column with `Edit` (opening
`EntityDialog` with the row) and `Delete` (opening `ConfirmDialog` for the
row). Omitting `create`, `update` or `remove` omits the corresponding button,
and omitting both `update` and `remove` omits the `Actions` column.

**Dialogs and refresh.** Saving in `EntityDialog` emits `saved` and calls
`refresh()` on the inner `Table`. Confirming in `ConfirmDialog` calls
`remove(row[idKey])`, emits `deleted` and refreshes the table; if `remove`
rejects with an `ApiError`, the dialog stays open and shows the server's
sentence in an `Alert`. `defineExpose` provides `refresh()`, `edit(record)`
and `askRemove(record)`.
