---
group: landing
purpose: A titled grid of quotes, each a figure with blockquote, author and avatar.
use: On a landing page to show quotes from people who use the product, with their name, role and avatar.
avoid: For a sequence of milestones over time, use Timeline; for capability cards with icons, use FeatureGrid.
adapts:
  input: Nothing changes unless an item sets `to`, which renders as a standalone Link reaching --weft-hit-min (44 px) on touch.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with its own block and inline padding (omitted inside a Page, Section or Container); in an app it runs full width.
  width: The inner Grid measures its own box (useWidth, before paint and on resize), reflowing by default from 1 column narrow (under 640 px) to 2 regular (under 1024 px) to 3 wide.
---
`items` are `{ quote, author, role, avatar, to }`. Each renders as a
`<figure>` holding a `<blockquote>` and a `<figcaption>` with a decorative
`Avatar` (`avatar` image, or initials from `author`), the author's name
(linked through `Link` when `to` is given) and `role`.

The section's `title` renders as an `h2` by default (`level` changes it to
2–6). `cols` takes a fixed column count or `{ narrow, regular, wide }`,
forwarded to `Grid`. The `item` slot replaces an item's quote and caption.
