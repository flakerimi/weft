---
group: forms
purpose: Controls joined into one, their borders meeting.
use: For an input with a button at its end, a unit before an amount, a row of segmented buttons.
avoid: For laying out separate fields, use Stack or Grid.
adapts:
  input: Nothing of its own; the controls in it keep their touch sizes, so on touch an md input and an md Button in a group are both 44 px tall.
  surface: Nothing changes.
  width: Nothing changes; a vertical group stacks its controls.
---
Put Inputs, InputNumbers and Buttons in the default slot. Neighbouring borders
overlap, only the group's outer corners are round, and the focused control is
drawn on top so its ring shows whole. `orientation` is `horizontal` or
`vertical`. Form controls in it take its `size` unless they set their own;
give Buttons in it the same `size`. It is a `role="group"`; give it an
`aria-label` when the controls need a shared name.
