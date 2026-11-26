---
group: data
purpose: Rows of data under column headers, sorted, chosen, paged and loaded from an array or from a server.
use: For records a person compares, sorts or acts on in bulk (customers, invoices, devices), from a plain array or a server speaking the wire format.
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

**Rows.** Give `rows` (an array), sorted and paged on the client (text in
the WeftProvider's `locale`, en-US without one, numbers and dates by
amount), or `fetch(query)`, which receives the wire format's list query
in its names, `{ page, per_page, sort, q, filter }` (each but `page` and
`per_page` is absent when there is none; `sort` is `name` or `-since`),
and answers the wire format's whole list, `{ items, total, page, per_page }`
(`page` and `per_page` are optional), so paging, sorting, searching and
filtering happen on the server. A list function from the generated client
fits as it is: `:fetch="api.products.list"`. Give the search text as
`search` (sent as `q`) and the filters as `filters` (`{ plan: 'pro' }`,
sent as `filter`); a change asks again. Clearing a filter is setting it to
an empty string or leaving it out: a filter with no value is not sent, and
`{ status: '' }` is the same query as `{}`. With `rows` and no `fetch`,
`search` and `filters` do nothing.
`tableSearch(query)` writes the query as the wire format's search
parameters: `page=2&per_page=20&sort=-since&q=ada&filter[plan]=pro`. It asks
once mounted (the server draws skeleton rows), again whenever the page, the
sort, the search or the filters change, ignores an answer a newer query
overtook, and shows a message with a Try again button (and emits `error`)
when the promise rejects; `refresh()` asks again. `rowKey` (a property name
or a function; default the row's `id`, else its place) keys the rows. The
data source is `useTableData`, exported for a view of its own.

**Sorting.** A sortable header holds a button: ascending, descending, then
as given. The sort is `v-model:sort` in the wire format. On the client it is
stable (equal rows keep their order both ways), numbers and dates by amount,
text numeric-aware and case-blind, empty values last. A press on a header
goes back to page 1, as does a new `search` or new `filters` (with `fetch`); a
parent that sets the page and the sort, the search or the filters together
keeps its page. The sorted header says `aria-sort`; the others say nothing (APG
sortable table).

**Selection.** `selectable` adds a checkbox per row and one in the header,
bound with `v-model:selected` (the rows' keys). The header box is ticked
when every row shown is chosen, indeterminate when some are; it chooses or
clears the rows shown (on this page). Shift+click on a row's box sets every
row from the last one clicked to it, in the order shown. Keys persist
across pages and sorts: a row chosen on page 1 stays chosen on page 3 and
after a new sort. So rows from `fetch` that can be chosen need keys of their
own, an `id` or `rowKey`; without one a row's key is its place on its page,
and development builds warn once.

**Paging.** `per-page` pages the rows with a Pagination under the table
(`v-model:page`), shown when there is more than one page; with `fetch` it
defaults to 20. A fetched page past the last one, as after the last row of
the last page is deleted, moves the page to the last one there is, which it
then asks for, once. The move goes through `update:page`, so a parent that binds the
page one way (`:page` with no handler) and keeps it gets that page shown
empty, with the server's total, and no request until the page, the query or
a refresh changes.

**States.** `loading` makes it `aria-busy` and draws skeleton rows (hidden
from assistive technology, a status saying Loading) until there are rows;
rows it already has stay, dimmed. With no rows the `empty` slot (or
`empty`, default "Nothing to show.") fills the body.

**Sticky header.** With `sticky` (on by default) the header stays in view.
With a `height` (or once windowed) the rows scroll inside the table's box
and the header sticks at its top. Without one the box clips rather than
scrolls once mounted, so the header sticks against the page as it scrolls,
below `--weft-table-sticky-top` (set it to a sticky page header's height);
but a table wider than its box must scroll sideways, and then its header
sticks only inside it: give it a `height`. Before it mounts (and on a page
with no island) the box scrolls, so a wide table is never cut off.

**Windowing.** Past 200 rows on a page it renders only the rows in view,
spacer rows standing for the rest (`useVirtualList`): every row is then
`row-height` px (default by size, at least 44 on touch), its text on one
line, the box 30rem tall unless `height` says otherwise, and the table
states `aria-rowcount` and each row `aria-rowindex`. The narrow cards are
not windowed; page long lists.

**Keyboard.** By default it is a plain `<table>` (caption, `th scope`), and
each control in it (sort buttons, checkboxes, controls in cells) is its own
tab stop. With `keyboard` it is an APG grid, one tab stop, so Tab enters and
leaves it in one step: arrows move between cells (mirrored right to left)
and stop at the edges, Home and End go along the row, Ctrl+Home to the first
header cell and Ctrl+End to the last cell of the last row, windowed or not.
The focused cell scrolls into view, and a windowed table keeps its row drawn
when the pointer scrolls it away, so focus is never lost. A sort button or a
checkbox stands in for its cell. Controls a cell slot draws leave the tab
order (`tabindex="-1"`); Enter or F2 on the cell gives them back and focuses
the first, keys pressed in it are its own, and Escape, or focus leaving the
cell, takes them out again. `caption` names the table (and the
list of cards); `hide-caption` keeps it for assistive technology only.
`labels` translate its words.
