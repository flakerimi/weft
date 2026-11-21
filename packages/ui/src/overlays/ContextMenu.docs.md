---
group: overlays
purpose: A menu for an area, opened in place of the browser's own by a right press, the context menu key or a long press.
use: For actions on an item that a power user expects on right press (a file, a row, a canvas), alongside a visible way to reach them.
avoid: As the only way to an action, since nothing shows that it is there; give the item a DropdownMenu button too.
adapts:
  input: On pointer it opens at the pointer. On touch a long press (600 ms, held still) opens it as a bottom sheet with large items; a finger that moves is scrolling and opens nothing.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: As DropdownMenu.
---
The default slot is the area; `items` are DropdownMenu's `MenuItem`s, and
the menu is a DropdownMenu, with the same keyboard, submenus and closing.
Opened with the keyboard (the context menu key or Shift+F10 on something
focused inside), it opens under that element. Focus returns to where it was.
Attributes go to the menu: give it an `aria-label`. `disabled` leaves the
browser's own menu in place.
