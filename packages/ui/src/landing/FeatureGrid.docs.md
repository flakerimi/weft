---
group: landing
purpose: A titled grid of capabilities, each with an icon, heading, text and link.
use: On a landing page to highlight what a product does, each capability in its own card or column.
avoid: For key numbers on a dashboard, use StatCard in a Grid; for tabular records, use Table.
adapts:
  input: Any item link (`to`, with or without `linkLabel`) renders as a standalone Link, reaching --weft-hit-min (44 px) on touch.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with its own block and inline padding (omitted inside a Page, Section or Container); in an app it runs full width.
  width: The inner Grid measures its own box (useWidth, before paint and on resize), reflowing by default from 1 column narrow (under 640 px) to 2 regular (under 1024 px) to 3 wide.
---
`items` are `{ title, description, icon, to, linkLabel, slot }`. The section's
`title` renders as an `h2` by default (`level` changes it to 2–6), and each
item's title renders one heading level deeper (`h3` under `h2`). `icon` draws
a Lucide icon in a disc tinted in `tone`. With `to` and `linkLabel`, a
standalone link sits under the description; with `to` alone, the item's title
links there as a standalone link.

`variant` is `card` (each item in a bordered surface box, the default) or
`plain` (unboxed items). `cols` takes a fixed column count or `{ narrow,
regular, wide }`, forwarded to `Grid`. The `item` slot replaces every item's
body, and a slot named by an item's `slot` replaces that item's body alone.
