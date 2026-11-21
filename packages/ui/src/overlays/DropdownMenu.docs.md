---
group: overlays
purpose: A menu of actions opened from a button, with groups, checkbox and radio items, links and submenus.
use: For the actions on one thing (a row's More button), a user menu, view options.
avoid: To choose a value for a form, use a select; for navigation between pages, links or navigation components.
adapts:
  input: On pointer it floats beside its trigger and a submenu opens beside its item, on hover too. On touch it is a bottom sheet with large items, 48 px tall with larger text, modal, and a submenu is a sheet above it, with a back item. The input can change while it is open; the focused item stays focused.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Floating, it is at least 10rem wide and stays within the room left; as a sheet, the viewport's width.
---
`items` is a list of `MenuItem`s, or a list of groups (an array of arrays),
drawn with separators between. An item has `label`, `icon` (a Lucide name),
`tone` (`danger`), `disabled`, `to` and `target` (a link), `children` (a
submenu) and `onSelect(event)`; `type: 'label'` is a group's heading,
`'separator'` a line, `'checkbox'` and `'radio'` items with `checked` and
`onUpdateChecked(checked)`. An item's `description` shows under its label
(and describes it), `kbds` shows its shortcut with Kbd (shown, not bound),
`loading` turns a spinner and keeps it from being chosen (a link loses its
address meanwhile, so no Ctrl-click or middle click follows it), and `color` is
`tone` under Nuxt UI's name. The `item` slot replaces an item's label.

It follows the WAI-ARIA APG Menu Button and Menu patterns. The trigger, the
default slot (one element), gets `aria-haspopup="menu"`, `aria-expanded`
and `aria-controls`, and the menu is labelled by it.

**Keyboard.** On the trigger: Enter, Space and ArrowDown open the menu on
its first item, ArrowUp on its last. In the menu: ArrowDown and ArrowUp move
(looping), Home and End go to the ends, typing jumps to the item whose label
starts so, Enter and Space choose. ArrowRight opens a submenu on its first
item, ArrowLeft closes it back to its item (mirrored right to left). Escape
closes one level; Tab closes the menu, as a sheet on touch too. Disabled items are reached but cannot
be chosen.

**Choosing** an item calls `onSelect`, closes every level and puts focus back
on the trigger, unless `onSelect` calls `event.preventDefault()`. A checkbox
item calls `onUpdateChecked(!checked)`, a radio item `onUpdateChecked(true)`.
A press outside closes it, returning no focus, from any level; while it is
open the page does not scroll. As a touch sheet it is modal: `aria-modal`,
focus trapped, the page behind inert. It fades out when it closes, unless
motion is reduced.

The menu renders into `#weft-overlays` once mounted; attributes go to it.
`anchor` positions it against another element or a point (ContextMenu).
