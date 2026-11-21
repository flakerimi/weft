---
group: overlays
purpose: Shows the toasts queued with useToast(), in a corner of the viewport, announced to assistive technology.
use: Once, in the app's layout; then call useToast().add from anywhere in the browser.
avoid: For a message tied to a form or a section, show it there instead.
adapts:
  input: Nothing changes in behaviour; toasts' buttons follow Button's touch sizes.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: At most 24rem wide, the viewport less a margin when narrower.
---
    const toast = useToast()
    toast.add({ title: 'Saved', description: 'Your draft is safe.', tone: 'success' })
    toast.add({ title: 'Deleted', actions: [{ label: 'Undo', onClick: restore }] })

`add` returns the toast with its `id`, for `update(id, patch)` and
`remove(id)`; `clear()` empties the queue and `toasts` reads it. A toast
leaves after its `duration`, by default the theme's `--weft-duration-toast`.

The Toaster is a region named "Notifications" (`label`) whose list is a
polite live region, so a new toast is announced without moving focus. While
the pointer is over it or focus is inside it, no toast's time runs. F8 moves
focus to it, to reach a toast's actions; when the focused toast goes, focus
stays in the region. `position` is a corner or the middle of the top or
bottom edge (`bottom-end` by default; start and end follow the text
direction), and `max` (5) toasts show at once, the newest.

It renders into the overlay root only once mounted. The queue is one per
page; on the server `add` does nothing.
