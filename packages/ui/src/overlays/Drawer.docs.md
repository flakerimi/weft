---
group: overlays
purpose: A sheet with a handle that a finger can drag away, from the bottom by default.
use: For a touch-first panel of options or details that the user dismisses with a swipe, as on a phone.
avoid: For a panel at the side of a large screen, use Sheet; for a decision, Dialog.
adapts:
  input: Dragging its handle or header towards its edge moves it with the finger; past a quarter of its size, or with a flick, it closes, short of that it springs back. Pointer users can drag too, or use Escape and the backdrop.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: As Sheet: a bottom or top drawer is the viewport's width and at most 85% of its height.
---
`side` is `bottom` (default), `top`, `start` or `end` (mirrored right to
left); `handle` shows the drag handle (default true); the close button is
off by default (`close` shows it). Otherwise it is a Sheet, with the same
slots, `v-model:open`, focus trap, scroll lock and closing by Escape and
the backdrop, focus back on the trigger. `dismissible="false"` turns off
dragging as well.

A press on a control in the header is that control's, never the start of a
drag.
