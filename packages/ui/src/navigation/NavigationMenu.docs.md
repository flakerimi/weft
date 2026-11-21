---
group: navigation
purpose: A site's or an app's main links, as a bar with dropdowns or as a sidebar column.
use: For the main navigation in a header (horizontal) or a sidebar (vertical).
avoid: For actions, use DropdownMenu; for a page's place in the site, Breadcrumb; for a whole app frame, AppShell, which draws one.
adapts:
  input: On touch every item is at least --weft-hit-min tall, and a dropdown opens as a bottom sheet (DropdownMenu's rule). On a narrow touch screen a horizontal menu is one menu button that opens the items as a vertical list in a Sheet from the start side.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Horizontal on a narrow touch screen it becomes the menu button and sheet; on a narrow pointer screen it stays a bar. Vertical, it fills its container.
---
`items` are `NavItem`s, or groups of them (an array of arrays) drawn apart:
`{ label, to, target, icon, active, disabled, badge, description, children,
defaultOpen, onSelect }`; `type: 'label'` is a heading over a vertical group.
An item with neither `to` nor `children` is a button that calls `onSelect`.
`label` names the landmark (`Main`); `menu-label` the narrow menu button.
It emits `select(item, event)` when a link is followed or a button pressed,
and the `item` slot replaces an item's label.

**Current page.** Items are Weft Links: `active` sets `aria-current="page"`;
unset, a link compares its address with the browser's, after mounting. An
item whose child is `active` is drawn as current too.

**Horizontal** (on pointer, and on wide touch screens): a row of links, and
of buttons that open their children as a DropdownMenu (APG Menu Button: the
children are `menuitem` links). Every item is in the tab order; Left and
Right also move between them (mirrored right to left), Home and End to the
ends.

**Vertical** (a sidebar): an item with children is a disclosure button
(`aria-expanded`, `aria-controls`) over a list rendered and hidden with
`hidden="until-found"`, so find-in-page opens it; it starts open with
`defaultOpen` or when a child is `active`. ArrowDown and ArrowUp move over
the visible items. `collapsed` draws icons alone: labels stay for assistive
technology and show in a Tooltip on pointer, and children open as a
DropdownMenu beside the item.
