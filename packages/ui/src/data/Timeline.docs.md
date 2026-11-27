---
group: data
purpose: Events in order on a line, each with its date, title, description and status.
use: For an order's progress, an account's history or a changelog: what happened, and when.
avoid: For steps a person moves through and can go back to, use Stepper; for a log to search and sort, use Table.
adapts:
  input: Nothing changes; it is not interactive.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: A horizontal timeline turns vertical when its own box is narrow, so the text keeps its width. CSS does it from `data-width`, which the page's pre-paint script sets, so a page that is never hydrated adapts too.
---
`items` are `{ date, title, description, icon, value }`, oldest first;
`reverse` lists the newest first. `date` is text as it should read ("Mar
4", "2 hours ago"). An item with an `icon` draws it in a disc; the others
are dots.

Give the current item with `v-model` (its `value`, or its place from 0):
the items before it are done, drawn in `tone` with their line, and said as
"Done:" before the title (translate with `labels`); the current one is
ringed and says `aria-current="step"`; the later ones are muted. With no
current item every item is drawn alike. The timeline only shows this; it
never changes it.

It is an ordered list; the dots and lines are hidden from assistive
technology. The `indicator`, `date`, `title` and `description` slots draw
an item's parts (`{ item, index }`). `orientation` is `vertical` (default)
or `horizontal`; `size` sets the dots and text.
