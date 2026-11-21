---
group: selection
purpose: A list of options always shown, to choose one or several from, with an optional filter field above it.
use: For a choice that deserves the room of a visible list (a few to a few thousand options), in a form or a side panel; with `filter` when the list is long.
avoid: When the list should open from a field, use Select or Combobox; for two to five options, RadioGroup or CheckboxGroup; for actions, DropdownMenu or CommandPalette.
adapts:
  input: Options are at least --weft-hit-min tall on pointer and 48 px on touch, with larger text; the pointer highlights the option under it.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container and scrolls past 18rem tall; labels and descriptions are cut with an ellipsis.
---
`items` are values (`'Apple'`) or objects (`{ label, value, description,
icon, disabled }`), or groups of them (an array of arrays), drawn with
separators between; an object with `type: 'label'` is its group's heading
and `type: 'separator'` a line. Without `value-key` an object is itself the
value; `label-key` and `description-key` read other keys (dotted paths
work), and `by` compares values by a key or a function. Bind the choice with
`v-model` (an array with `multiple`).

It follows the WAI-ARIA APG Listbox pattern. The list is one tab stop and
keeps focus; `aria-activedescendant` names the active option, drawn
highlighted. Inside a FormField the list is named by the label
(`aria-labelledby`) and described by the description and error.

**Keyboard.** ArrowDown and ArrowUp move (with `loop` they wrap), Home and
End go to the ends, Page Down and Page Up move ten, typing jumps to the
option whose label starts so; disabled options are skipped. Enter and Space
choose; with `multiple` they toggle. Focusing the list makes the chosen
option active, else the first.

**Filter.** With `filter` a search field above the list takes focus instead
(`role="combobox"`, controlling the list); typing filters by label (or
`filter-fields`), ignoring case and accents, the closest matches first, and
the arrows and Enter work from the field. `v-model:search-term` binds the
search; `ignore-filter` leaves the filtering to the app. `create-item` offers
what was typed as a new option, emitting `create`. When nothing is (or
matches) the `empty` slot shows, by default "No results", in a status that
assistive technology announces; `loading` shows a spinner there and sets
`aria-busy`.

**Long lists.** Past 200 options only the rows in view are drawn (each of a
fixed height, from the size), over a spacer as tall as the list; options
carry `aria-setsize` and `aria-posinset`, and the active option is always
drawn, so `aria-activedescendant` never names a missing element.
`virtualize` forces it either way.

With a `name` (or its FormField's) the value submits in hidden inputs, one
per value. The slots `item`, `item-leading`, `item-label`, `item-trailing`
and `create-item-label` replace parts of an option. Select, Combobox and
CommandPalette draw their lists with it.
