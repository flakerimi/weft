---
group: navigation
purpose: Buttons to move through the pages of a long list.
use: Under a table or a list of results that loads a page at a time.
avoid: For a feed a reader scrolls, load more as they reach the end; for steps of a task, use Stepper.
adapts:
  input: Every button is at least --weft-hit-min tall on touch (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: When the viewport is narrow it shows at most one page each side of the current one and drops the first and last buttons; the row wraps when it must.
---
Bind the current page with `v-model:page` (from 1). `total` is the number
of items and `items-per-page` the size of a page. `sibling-count` (2) pages
show on each side of the current one; `show-edges` always shows the first
and last, with an ellipsis for each gap, keeping the row the same length.
`show-controls` (on) adds first, previous, next and last. `to(page)` makes
every button a link to that page's address. `variant` and `tone` draw the
other pages, `active-variant` and `active-tone` the current one; `labels`
translate the words for assistive technology.

It is a `nav` landmark (`Pagination`). The current page says
`aria-current="page"`; each page button is named `Page n`; the ellipses are
hidden from assistive technology. The controls at an end are disabled, and
`disabled` disables every button. **Keyboard:** each is a button (or a
link), in the tab order; Enter and Space press it.

`pageRange(page, count, siblings, edges)` is the same arithmetic, exported.
