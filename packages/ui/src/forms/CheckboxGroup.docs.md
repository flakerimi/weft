---
group: forms
purpose: Several checkboxes for one list of chosen values.
use: For choosing any number of a few options, all in view (toppings, notification channels).
avoid: For exactly one choice, use RadioGroup; for many options, use a multiple select.
adapts:
  input: Each row is at least 44 px tall on touch, and each box's hit area 44 px square, so rows do not steal each other's presses.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Horizontal groups wrap onto more rows when narrow.
---
Bind the list with `v-model`. Give the options as `items` - values, or
`{ value, label, description, disabled }` - or put Checkboxes with a `value`
in the default slot. A ticked value is added at the end of the list.

It is a `<fieldset>`: `legend` names it, or, inside a FormField, the field's
label does through `aria-labelledby`; the field's hint and error describe the
group, and an error marks it `aria-invalid`. The boxes keep ids of their own.
`disabled` disables every box. Tab visits each box, as native checkboxes do.
