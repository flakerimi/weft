---
group: forms
purpose: One choice among a few options, all in view.
use: For a choice of two to five options the reader should compare at a glance (a plan, a shipping speed).
avoid: For many options, use a select; for on or off, use Switch or Checkbox.
adapts:
  input: Each row is at least 44 px tall on touch, and each radio's hit area 44 px square.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Horizontal groups wrap onto more rows when narrow.
---
Bind the chosen value with `v-model`; give the options as `items` - values,
or `{ value, label, description, disabled }`.

**Keyboard** (the WAI-ARIA radio group pattern): the group is one tab stop,
on the chosen radio or else the first. Down and Right move to the next radio
and choose it, Up and Left to the previous, wrapping at the ends and
skipping disabled radios; in a right-to-left document Left and Right swap.
Focus arriving by Tab or a click chooses nothing by itself; Space chooses
the focused radio.

The radios are native, sharing `name` (one is generated when not given), so
they post with a plain form. The group is `role="radiogroup"`, named by
`legend` or by its FormField's label, described by the field's description, hint and
error, and `aria-required` when the field (or `required`) asks.
