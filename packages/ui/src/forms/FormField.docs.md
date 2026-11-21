---
group: forms
purpose: The label, hint, error and required marker around one form control.
use: Around every form control that needs a visible label, inside a Form or not.
avoid: For a Checkbox or Switch whose own label says everything, the control alone is enough.
adapts:
  input: Nothing of its own; the control inside it adapts.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; it fills the width of its container.
---
The label sits over the control and is `for` it: the control takes the
field's id (set `id` to choose it). A group control (RadioGroup,
CheckboxGroup, PinInput, a range Slider) is named through `aria-labelledby`
instead. `hint` sits under the label; `error` sits under the control. Both
describe the control through `aria-describedby`, and an error marks it
`aria-invalid`. `error: true` marks it invalid without a sentence (or with
the `error` slot). `required` draws an asterisk, hidden from assistive
technology, and sets `aria-required` on the control.

**In a Form,** `name` links the field to the Form: it shows the Form's error
for that name, it is required when the schema says so, and leaving or
changing the control lets the Form check it again. An `error` given to the
field wins over the Form's.
