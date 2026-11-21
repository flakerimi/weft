---
group: selection
purpose: A search field over grouped commands, found by fuzzy matching and run from the keyboard, with recent commands and pages of sub-commands.
use: For a Cmd+K launcher in a Dialog, or a search-first menu of everything an app can do or go to.
avoid: For a value in a form, use Combobox; for a handful of actions on one thing, DropdownMenu.
adapts:
  input: Commands are at least --weft-hit-min tall, so 44 px on touch; inside a Dialog it is full-screen when narrow, as the Dialog is.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills its container (a Dialog's body); the list scrolls past 24rem, or 60% of the viewport's height.
---
`groups` are `{ id, label, items }`; a command has `label`, and may have
`id`, `icon`, `prefix`, `suffix`, `description`, `keywords`, `kbds` (its
shortcut, shown with Kbd; shown, not bound), `disabled`, `loading`, `to` and
`target`, `children` and `onSelect(event)`. Running a command calls its
`onSelect`, then emits `select`, then follows `to` unless `onSelect`
prevented the event's default. A command with `children` opens them as a
page of their own, headed by its label, its `placeholder` in the field, with
a back button; Backspace in the empty field goes back.

**Search.** Typing matches fuzzily: each word's letters in order, gaps
allowed, ignoring case and accents, word starts and runs of letters scoring
higher; descriptions, suffixes and keywords count for half. The matched
letters of each label are marked. Matches come best first, at most `limit`
(12) per group, and the groups by their best match unless
`preserve-group-order`. A group with `ignoreFilter` shows its items whatever
the search, and `postFilter(term, items)` has the last word on a group. When
nothing matches the `empty` slot shows, by default "No results".

**Keyboard.** The field keeps focus (`role="combobox"`, its list always
shown, `aria-activedescendant` naming the active command, which is also the
selected one): ArrowDown and ArrowUp move across groups, Page Down and Page
Up move ten, skipping disabled and loading commands; Enter runs the active
command; Home and End stay with the text.

**Recent.** With `recent` (5, or a number) the commands run last are listed
first, under `recent-label`, while the search is empty; with `recent-key`
they are kept in localStorage (`weft:recent:<key>`), read once mounted.

**In a Dialog.** Put it in a Dialog's `body` slot: the Dialog's focus trap
moves focus into the field when it opens, Escape closes the Dialog and focus
returns to its opener; close the Dialog on `select`. `close` adds a close
button to the field, emitting `update:open` with false. The example
`in-dialog` opens one with Cmd+K or Ctrl+K.

The slots `item-label` and `item-trailing` replace or add to a command's
parts, and `footer` goes under the list.
