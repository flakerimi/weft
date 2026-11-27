---
group: forms
purpose: Text over several lines, which can grow with what is typed.
use: For a message, a description, a note; with autoresize where the length varies.
avoid: For one line, use Input; for formatted text, use an editor.
adapts:
  input: It is never shorter than an md control, so at least 44 px on touch whatever its rows and size.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container.
---
Bind the text with `v-model` (`.trim` and `.lazy` work). `rows` sets how many
lines show (3). With `autoresize` it grows as text is added, from `rows` up to
`maxrows` (0, the default, for no limit) and then scrolls, and it shrinks
back when text is removed; it cannot be resized by hand then. A template ref
has `focus()`, `resize()` (to fit after being shown) and `textarea`.

In a FormField it takes the field's id, name, description, error and required
state, as Input does; outside a FormField, with no `id` prop, it takes a
generated id. Attributes go to the `<textarea>`.
