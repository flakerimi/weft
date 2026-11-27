---
group: layout
purpose: Lays its children out in equal columns, as many as its own width allows.
use: For tiles, cards and stat blocks that should reflow from one column to several as the space grows.
avoid: For a single column or row, use Stack; for tabular data, use a table.
adapts:
  input: Nothing changes.
  surface: Nothing changes.
  width: Adapts in CSS first from the grid's own container width (@container), not the window, so a grid in a side panel or static page stays narrow on a large screen; by default 1 column narrow (under 640 px), 2 regular (under 1024 px) and 3 wide.
---
`cols` is a fixed number or one per width class, `{ narrow, regular, wide }`.
A width left out falls back to the nearest smaller one given, else to the
default for it.
