---
group: overlays
purpose: One short message that appears for a moment, with an optional action: saved, sent, undone.
use: Rendered by Toaster from useToast(); render one yourself only inside a live region of your own.
avoid: To show a toast, call useToast().add and put a Toaster in the layout; for a message that must stay, use an inline alert.
adapts:
  input: Nothing changes in behaviour; its close button's hit area is at least 44 px on touch.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills its Toaster's column.
---
`title`, `description`, `icon` (a Lucide name), `tone` (a bar at the inline
start), `actions` (buttons, each closing the toast after it runs) and a
close button (× drawn inline; `close="false"` hides it). The default,
`leading` and `actions` slots replace the text, icon and actions.

It emits `close` when it should go: after `duration` ms (by default the
theme's `--weft-duration-toast`, 4000 ms in the neutral theme; 0 stays until
closed), when closed, acted on, or on Escape while focus is in it. Its time
stands still while `paused`, and resumes with what it had left.
