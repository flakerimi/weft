---
group: navigation
purpose: A site's or an app's main links, as a bar whose groups open as lists, or as a sidebar column.
use: For the main navigation in a header (horizontal) or a sidebar (vertical).
avoid: For actions, use DropdownMenu; for a page's place in the site, Breadcrumb; for a whole app frame, AppShell, which draws one.
adapts:
  input: On touch every item is at least --weft-hit-min tall, in the bar and in an open list. On a narrow touch screen a horizontal menu is one menu button that opens the items as a column in a Sheet from the start side. A change of input with a list or the sheet open closes it and keeps focus on the same item in the new form.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Horizontal on a narrow touch screen it becomes the menu button and sheet; on a narrow pointer screen it stays a bar, scrolling sideways when it overflows. Before mounting CSS picks the form from the viewport. Vertical, it fills its container.
---
`items` are `NavItem`s, or groups of them (an array of arrays) drawn apart:
`{ label, to, target, icon, active, disabled, badge, description, children,
defaultOpen, onSelect }`; `type: 'label'` is a heading over a vertical group.
An item with neither `to` nor `children` is a button that calls `onSelect`.
`label` names the landmark (`Main`); `menu-label` the narrow menu button.
It emits `select(item, event)` when a link is followed or a button pressed
(in the narrow sheet too), and the `item` slot replaces an item's label.

It follows the WAI-ARIA APG **Disclosure Navigation** pattern: links stay
links, and an item with children is a disclosure button (`aria-expanded`,
`aria-controls`) that opens a list of links. There is no `menu` role.

**Current page.** Items are Weft Links: `active` sets `aria-current="page"`;
unset, a link compares its address with the browser's, after mounting. An
item whose child is the current page is drawn as current too.

**Forms.** Once mounted the nav says which it draws in `data-form`:

- `bar` (horizontal, on pointer and on wide touch screens): a row of links
  and disclosure buttons; an open list floats under its button (fixed, so a
  scrolling parent does not clip it). One list is open at a time; it closes
  when focus leaves it, on a press outside, when Escape is pressed (focus
  back on its button) and when one of its links is chosen.
- `sheet` (horizontal on a narrow touch screen): the menu button and Sheet.
- `column` (vertical): a list opens below its button, rendered and hidden
  with `hidden="until-found"`, so find-in-page opens it. A group starts open
  with `defaultOpen` or when it holds the current page (by `active`, or by
  the browser's address once mounted), and follows new `items`. `collapsed`
  draws icons alone: labels stay for assistive technology and show in a
  Tooltip on pointer, and a group's list floats beside its button.

Before mounting there is no `data-form`: the server renders the bar and the
menu button, and CSS shows the menu button in place of the bar on a narrow
touch viewport (`max-width: 639.98px` and `pointer: coarse`).

**Keyboard.** Every item is in the tab order. In the bar, Left and Right
also move between the items (mirrored right to left), Home and End to the
ends; ArrowDown on a disclosure button opens its list and focuses its first
link (ArrowUp its last). In a column, ArrowDown and ArrowUp move over the
visible items; in a collapsed column the forward arrow opens a list. In an
open list ArrowDown and ArrowUp move (wrapping), Home and End go to the
ends, and Escape closes it back to its button. Enter and Space press a
button, as native buttons do.
