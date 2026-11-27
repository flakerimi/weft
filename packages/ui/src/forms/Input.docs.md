---
group: forms
purpose: A one-line text field that can hold an icon, unit or button beside the text.
use: For a short answer typed in full - a name, an email address, a search - bound with v-model, usually inside a FormField.
avoid: For text over several lines, use Textarea; for a number stepped up and down, use InputNumber; for a choice from a known list, use a select or RadioGroup.
adapts:
  input: An md input is max(--weft-control-md, --weft-hit-min) tall, so 44 px on touch, the same height as an md Button beside it; sm and lg keep their sizes, and a pseudo-element behind the field grows the hit area to 44 px.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container; put it in a Grid or a FieldGroup to share a row.
---
Bind the text with `v-model`; `v-model.trim`, `.number` and `.lazy` work as
on a native input (`type="number"` also gives a number, or `null` when it is
empty). Attributes such as `placeholder`, `autocomplete`, `inputmode` and
`aria-label` go to the `<input>`; `class` and `style` go to the box.

**Leading and trailing.** `icon` and `trailing-icon` draw Lucide icons, hidden
from assistive technology; the `leading` and `trailing` slots replace them
with anything (a unit, a clear button). A press on the box outside the text
focuses the text; a button in the box keeps its own press.

**Loading** puts a spinner after the text and sets `aria-busy`; the field
stays editable (a lookup runs as it is typed).

**In a FormField** the input takes the field's id (so the label points to
it), its name, `aria-describedby` (the description, the hint and the error; an
`aria-describedby` of its own comes first), `aria-invalid` and
`aria-required`. Standalone, with no `id` or `name`, it takes a generated id.
In a FieldGroup it takes the group's size.

**For other components:** a template ref's `focus()` focuses the text, and
`input` is the element; Select, Combobox and DatePicker build on this box.
