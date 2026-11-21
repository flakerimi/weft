---
group: data
purpose: Rows of data under column headers, sorted, chosen, paged and loaded from an array or from a server.
use: For records a person compares, sorts or acts on in bulk (customers, invoices, devices), from a plain array or a Go resource speaking the wire format.
avoid: For a few label–value pairs about one thing, use a description list or StatCard; for items a person reads one by one, use a list of Cards; for a hierarchy, use Tree.
adapts:
  input: On touch every row is at least --weft-hit-min (44 px) tall, and its checkbox takes a 44 px press (Checkbox's rule).
  surface: Nothing changes in behaviour; `size` sets the row height and text follows the surface's type scale.
  width: When its own box is narrow (under 640 px, even in a side panel on a wide screen) each row becomes a card, a list item of label–value pairs from the same columns, with a Select all checkbox and the sort buttons above the cards; wide again, it is a table again.
---
`columns` is a typed array: `{ key, label, sortable, align, width,
accessor, format, rowHeader }`. A cell shows `accessor(row)` (default
`row[key]`) as text, or `format(value, row)`; the `<key>-cell` slot draws it
instead (`{ row, value, index, column }`), and `<key>-header` the header. A
`rowHeader` column's cells are `th scope="row"`, and its value names each
row's checkbox ("Select Ada").

**Rows.** Give `rows` (an array), sorted and paged on the client, or
`fetch(query)`, which receives the wire format's `{ page, perPage, sort }`
(sort is `name` or `-since`, absent when unsorted) and answers `{ items,
total }`, so paging and sorting happen on the server. `tableSearch(query)`
writes those as search parameters (`page=2&perPage=20&sort=-since`) for a
Go `weft.Resource`. It asks once mounted (the server draws skeleton rows),
again whenever the page or the sort changes, ignores an answer a newer query
overtook, and shows a message with a Try again button (and emits `error`)
when the promise rejects; `refresh()` asks again. `rowKey` (a property name
or a function; default the row's `id`, else its place) keys the rows.

**Sorting.** A sortable header holds a button: ascending, descending, then
as given. The sort is `v-model:sort` in the wire format. On the client it is
stable (equal rows keep their order both ways), numbers and dates by amount,
text numeric-aware and case-blind, empty values last. A new sort goes back
to page 1. The sorted header says `aria-sort`; the others say nothing (APG
sortable table).

**Selection.** `selectable` adds a checkbox per row and one in the header,
bound with `v-model:selected` (the rows' keys). The header box is ticked
when every row shown is chosen, indeterminate when some are; it chooses or
clears the rows shown (on this page). Shift+click on a row's box sets every
row from the last one clicked to it, in the order shown.

**Paging.** `per-page` pages the rows with a Pagination under the table
(`v-model:page`), shown when there is more than one page; with `fetch` it
defaults to 20.

**States.** `loading` makes it `aria-busy` and draws skeleton rows (hidden
from assistive technology, a status saying Loading) until there are rows;
rows it already has stay, dimmed. With no rows the `empty` slot (or
`empty`, default "Nothing to show.") fills the body.

**Sticky header and windowing.** The rows scroll inside the table's own box
under a sticky header (`sticky`, on by default) when `height` limits it.
Past 200 rows on a page it renders only the rows in view between two spacer
rows (`useVirtualList`): every row is then `row-height` px (default by size,
at least 44 on touch), its text on one line, the box 30rem tall unless
`height` says otherwise, and the table states `aria-rowcount` and each row
`aria-rowindex`. The narrow cards are not windowed; page long lists.

**Keyboard.** By default it is a plain `<table>` (caption, `th scope`), and
each control in it (sort buttons, checkboxes, controls in cells) is its own
tab stop. With `keyboard` it is an APG grid: one tab stop; arrows move
between cells (mirrored right to left), Home and End along the row,
Ctrl+Home and Ctrl+End to the first and last cell; a sort button or a
checkbox stands in for its cell; Enter or F2 moves into a cell's control and
Escape back to the cell, and keys pressed in that control are its own.
Controls in cells stay in the tab order. `caption` names the table (and the
list of cards); `hide-caption` keeps it for assistive technology only.
`labels` translate its words.
