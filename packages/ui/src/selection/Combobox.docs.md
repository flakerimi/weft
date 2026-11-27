---
group: selection
purpose: A text field that searches a list as you type to choose one or several items.
use: For a choice from a list people search (a person, a city, a tag), from many options or ones loaded from the server, or where a typed value may become a new option; Nuxt UI's InputMenu, and as a button its SelectMenu.
avoid: For a short list nobody searches, use Select; for free text with no list, Input; for running commands, CommandPalette.
adapts:
  input: On pointer the list floats under the field, focus staying in it. On touch the field only opens the list (it is read-only there, so no keyboard opens for it) as a modal bottom sheet with a search field at the top, which takes focus and filters; options are 48 px tall. The input can change while it is open; the search and the active option stay, and focus moves between the field and the sheet's search field.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The field fills its container's width, growing taller to hold tags with multiple; the floating list stays within the room left; the sheet is the viewport's width.
---
`items` are values or objects (`{ label, description, icon, disabled }`), or
groups of them, as in Listbox; without `value-key` an object is itself the
value. Bind the choice with `v-model` (an array with `multiple`), the search
with `v-model:search-term` and the list with `v-model:open`.

It follows the WAI-ARIA APG Combobox pattern with a listbox popup and list
autocomplete: the field has `role="combobox"`, `aria-autocomplete="list"`,
`aria-expanded`, and, while open, `aria-controls` and
`aria-activedescendant`. Inside a FormField the label is `for` the field
and names the list. The field's text is the search, so it submits nothing;
with a `name` (or its FormField's) the value submits in hidden inputs.

**Keyboard.** Typing opens the list and filters it, the first match active.
ArrowDown opens it on the chosen option (Alt+ArrowDown without moving),
ArrowUp on the last; open, the arrows and Page Up and Page Down move,
skipping disabled options. Enter chooses (and, while the list is open, never submits the form),
Escape closes, and a second Escape clears the text and the value;
Alt+ArrowUp and Tab close. Home and End move the caret. Leaving the field
puts back the chosen label. With `multiple` the chosen values are tags
before the text, each with a × (named "Remove …"); Backspace in the empty
field removes the last, and the list stays open as you choose.

**Searching.** The list keeps the options whose label (or `filter-fields`)
contains the search, ignoring case and accents, the closest first; the
`empty` slot shows when none does, by default "No results". `create-item`
offers what was typed as an option, emitting `create`: when nothing matches,
or `'always'` unless an option is the search, at the `bottom` or `top`.

**Async.** `load(term, { signal })` returns the items for a search: it runs
when the list opens and, `debounce` ms (200) after typing stops, for each
new search; the latest answer is shown as given, an older one arriving late
is dropped, and the signal aborts it. From the keystroke on the field shows a spinner and
the list is `aria-busy`, saying "Loading…" in place of an empty list; Enter
chooses nothing meanwhile, as the options shown are for the search before
(the `loading` prop alone does not hold Enter back). Closing the list stops
it: the timer goes, a load on its way is aborted, nothing stays busy.
A failure emits `error`, drops the results (they answered another search),
and shows "Couldn’t load results" (the `error` slot, given the error) until
the next load. Or filter yourself: bind `v-model:search-term`, set
`ignore-filter` and `loading`, and pass the items (Nuxt UI's way).

`trigger="button"` makes it a button (Nuxt UI's SelectMenu) showing the
choice, whose list has the search field at the top: the button has
`aria-haspopup="dialog"`, the search takes focus as the list opens, and
choosing returns focus to the button. `mode="autocomplete"` binds the text
itself, the list suggesting completions. `clear` adds a clear button while
there is a value (emits `clear`); `open-on-focus` and `open-on-click` open
the list sooner. `required` sets `aria-required` (a Form's schema or a
FormField can too); the field gets no native `required`, which would read
the search, not the value. `disabled` blocks typing, the chevron and every
key.
