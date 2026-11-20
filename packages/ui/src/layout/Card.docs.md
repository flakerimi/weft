---
group: layout
purpose: A bordered surface holding related content, with an optional header, actions and footer.
use: To group a few things that belong together, a plan, a summary, a small form, so they read as one unit.
avoid: For a page region with a heading, use Section; for content that needs a focused task, use a dialog.
adapts:
  input: Nothing changes; the card is not interactive itself.
  surface: Nothing changes.
  width: Nothing changes; header actions wrap under the title when they do not fit.
---
`padding` takes 0, 2, 3, 4 or 6 from the spacing scale; use 0 for edge-to-edge
content such as an image or a table. `as` renders another element, such as
`article` or `li`.
