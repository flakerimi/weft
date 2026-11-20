---
group: layout
purpose: Lays its children out in a column or a row with a gap from the spacing scale.
use: For any run of things one after another, a form's fields, a toolbar's buttons, a card's lines, instead of margins or a wrapper with its own CSS.
avoid: For columns that should reflow with the space available, use Grid; for a page or a titled part of one, use Page or Section.
adapts:
  input: Nothing changes; hit targets are the children's own.
  surface: Nothing changes; the gap scale is the same spacing tokens on landing and app.
  width: Nothing changes; a row stays a row, so give it wrap when its children may not fit a narrow container.
---
`gap` takes the spacing scale (0, 1, 2, 3, 4, 6 or 8, from `--weft-space-*`).
`align` and `justify` work along the cross and main axes as in flexbox, and
`as` renders another element, such as `ul` or `nav`, for the right semantics.
