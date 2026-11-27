---
group: forms
purpose: A box to tick on its own or in a CheckboxGroup, with indeterminate support.
use: For a yes or no that is sent with a form (accept the terms), or a choice among several that are all allowed.
avoid: For a setting that takes effect at once, use Switch; for one choice among several, use RadioGroup.
adapts:
  input: The box keeps its drawn size; the native input lies over it at least --weft-hit-min square, and each row is at least --weft-hit-min tall, so on touch it takes a 44 px press without neighbours' hit areas overlapping.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; a long label wraps beside the box.
---
Bind `v-model` to `true`, `false` or `'indeterminate'`. Indeterminate (some
of what it stands for is chosen) draws a dash and is announced as mixed; a
press ticks it. A parent that does not take a change leaves it as bound.

It is a native checkbox, so Space toggles it, its label toggles it, and it
posts `value` (default `on`) under `name` with a plain form. `label` (or the
default slot) is beside the box; `description` sits under the label and
describes the box. Focus rings the drawn box.

In a CheckboxGroup a Checkbox with a `value` is ticked while that value is in
the group's list. In a FormField it takes the field's id, so the field's label
also names it.
