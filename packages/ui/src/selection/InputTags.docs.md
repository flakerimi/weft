---
group: selection
purpose: A text field that turns what is typed into tags, each with a ×, for a list of short free values.
use: For keywords, email addresses, labels: values typed rather than chosen from a list, several of them.
avoid: When the values come from a known list, use Combobox with multiple; for one value, Input.
adapts:
  input: The field is at least max(--weft-control-md, --weft-hit-min) tall, so 44 px on touch, and each × has a hit area of at least --weft-hit-min.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills its container's width and grows taller as tags wrap onto more lines.
---
Bind the tags with `v-model` (an array). Enter adds the typed text, trimmed,
as a tag (and does not submit the form; Enter in an empty field does), and
so does typing the `delimiter` (a comma; a pattern works). `add-on-paste`
splits a paste into tags, `add-on-tab` adds on Tab, `add-on-blur` when focus
leaves. A duplicate is refused unless `duplicate`, and so is a tag past
`max`: both emit `invalid` and the text stays. `convert-value` turns the text
into the tag's value and `display-value` the value into the tag's text.

The tags are a list (`role="list"`) before the text, each a list item with a
× button named "Remove …", outside the tab order. Backspace in the empty
field, or ArrowLeft at its start (ArrowRight right to left), moves focus onto
the last tag; the arrows move between tags and back to the field; Backspace
or Delete takes out the focused tag, focus going to its neighbour or the
field. Inside a FormField the text field takes the field's id, description,
error and required mark; with a `name` (or its FormField's) the tags submit
in hidden inputs, one each. `disabled` blocks typing and removing.
