---
group: forms
purpose: On or off, taking effect at once.
use: For a setting that applies as soon as it changes (Wi-Fi, notifications).
avoid: For a choice sent with a form (accept the terms), use Checkbox.
adapts:
  input: The track keeps its drawn size; its hit area is at least 44 px square on touch, and the row at least 44 px tall.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; a long label wraps beside the track.
---
Bind with `v-model`. It is a native checkbox with `role="switch"`, so Space
and its label toggle it, assistive technology reads it as on or off, and it
posts `value` under `name` while on.

**Loading** (the change is being saved) turns a spinner in the thumb, sets
`aria-busy` and `aria-disabled`, and blocks a press, Space and the label
alike, while it stays focusable. **Disabled** is native.

The thumb slides to the inline end when on, so to the left in a right-to-left
document.
