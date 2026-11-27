---
group: selection
purpose: A button showing the chosen value that opens a list to pick one or several.
use: For a choice from a known list in a form (a country, a status, a size), where the list is too long for radio buttons and nobody needs to type to find an option.
avoid: When people search the list or type their own value, use Combobox; for two to five options always in view, RadioGroup; for actions, DropdownMenu.
adapts:
  input: On pointer the list floats under the button, at least as wide, focus staying on the button. On touch it is a modal bottom sheet (focus trapped, the page behind inert) with the field's label as its title and a search field at the top, which takes focus and filters; options are 48 px tall. The input can change while it is open; the active option and the choice stay, and focus moves to the button or the search field.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The button fills its container's width; the floating list stays within the room left, flipping above when there is none below; the sheet is the viewport's width.
---
`items` are values or objects (`{ label, value, description, icon,
disabled }`), or groups of them, with `type: 'label'` headings and
`type: 'separator'` lines, as in Listbox. An object's value is read at
`value` (`value-key` names another key; `value-key=""` makes the object
itself the value). Bind the choice with `v-model` (an array with `multiple`,
which shows the chosen labels joined by commas and keeps the list open),
and the list with `v-model:open`.

It follows the WAI-ARIA APG Select-Only Combobox pattern: a `button` with
`role="combobox"`, `aria-haspopup="listbox"`, `aria-expanded` and, while
open, `aria-controls` and `aria-activedescendant`. Inside a FormField the
label is `for` the button and names the list; the description and error
describe the button, and an error marks it `aria-invalid`.

**Keyboard.** Closed: ArrowDown, ArrowUp, Enter and Space open the list on the
chosen option (else the first), Home and End on the first and the last, and
typing opens it on the option whose label starts so. Open: the arrows,
Home, End, Page Up and Page Down move, skipping disabled options; typing
jumps; Enter and Space choose and close (with `multiple` they toggle);
Alt+ArrowUp chooses and closes; Tab chooses and moves on; Escape closes
without choosing. Focus stays on the button throughout (on touch, on the
sheet's search field, and back on the button when it closes).

A press on the button toggles the list; a press outside closes it, and so
does focus leaving. `disabled` blocks the press and every key (the button is
natively disabled); `loading` draws a spinner in place of the chevron and
sets `aria-busy`. `search-input="false"` leaves the touch sheet's search
field out; the list then takes focus itself. Choosing the value already
chosen emits no `change`. With a `name` (or its FormField's) the value submits in
hidden inputs. Past 200 options only the rows in view are drawn, the active
one always. The `default` slot draws the value on the button, `leading`
and `trailing` its ends, and `item`, `item-leading`, `item-label`,
`item-trailing` and `empty` the list's parts.
