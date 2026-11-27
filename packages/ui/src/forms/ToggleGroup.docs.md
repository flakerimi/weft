---
group: forms
purpose: A row of toggle buttons, one pressed or none, or any number with `multiple`.
use: For a small set of choices shown as buttons in a toolbar: text alignment, a view (list or grid), the styles of a selection.
avoid: For a choice in a form with labels, use RadioGroup or CheckboxGroup; for switching panels, use Tabs; for more than five or six options, use Select.
adapts:
  input: On touch every button is at least --weft-hit-min tall (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: A horizontal group wraps onto more lines when its box is narrow (unless attached).
---
`items` are `{ label, value, icon, trailingIcon, ariaLabel, tone, variant,
disabled }` (a value defaults to the item's place). Bind `v-model` to the
pressed value (null for none), or, with `multiple`, to the list of pressed
values. Pressing the pressed item of a single group releases it.

It is a `role="group"` (give it an `aria-label`, or, inside a FormField, it
takes the field's id and the field's label names it through
`aria-labelledby`) of buttons that say `aria-pressed`. **Keyboard:** the group is one tab stop, which rests on the
pressed item; the arrow keys move along it (Left and Right, mirrored right
to left; Up and Down when `orientation="vertical"`), wrapping round unless
`loop` is off; Home and End go to the ends; disabled items are passed by;
Enter and Space press.

`variant` and `tone` (`ghost`, `neutral`) draw the buttons not pressed,
`active-variant` and `active-tone` (`soft`, `primary`) the pressed ones.
`attached` joins them into one bar. With a `name` each pressed value posts
with a plain form. `disabled` disables every button. The default slot draws
a button's label (`{ item, index, pressed }`).
