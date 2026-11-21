---
group: overlays
purpose: Content that floats beside a trigger, opened by a press: filters, a short form, details.
use: For a small panel of controls or information tied to one button, which the user opens, uses and closes.
avoid: For a list of actions, use DropdownMenu; for a short hint on hover, Tooltip; for a task that needs the user's full attention, Dialog.
adapts:
  input: On pointer it floats beside the trigger and is not modal. On touch the same content is a bottom sheet over a backdrop, modal, with `title` as its heading. The input can change while it is open; what is inside keeps its state and focus.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Floating, it stays within the room left on its side, flipping and sliding to stay in view; as a sheet it is the viewport's width.
---
The default slot is the trigger, one element (a Button); a press on it
toggles the popover, and it carries `aria-haspopup="dialog"`,
`aria-expanded` and `aria-controls`. The `content` slot is what it shows,
and gets `close()`. Bind `v-model:open` to open it from code; `anchor`
positions it against another element or a point.

The content renders into `#weft-overlays` at the end of the body (created
when first needed), only once mounted, so the server renders none of it. It
is a `role="dialog"` named by `title` (or an `aria-label`); attributes go to
it.

**Focus.** On open, focus moves to the first tabbable element inside, or to
the content itself (`auto-focus="false"` keeps it where it is, as a combobox
does). On pointer it is not modal: Tab can leave, and focus leaving closes it
(unless it went to an overlay opened from inside). With `modal`, or on touch,
focus is trapped and the page does not scroll.

**Closing.** Escape and a press outside close it (`dismissible="false"`
turns both off); a press on its trigger toggles it shut. Focus returns to
the trigger, except after a press outside, which put the user's attention
elsewhere. On touch a press on the backdrop closes it and does return focus.

**Hover mode.** `mode="hover"` opens it too while the pointer is over the
trigger or the content (after `open-delay`, 100 ms) and closes it when the
pointer has left both (after `close-delay`, 200 ms), without moving focus;
a press still toggles it, and on touch it is pressed as ever. `arrow` draws
an arrow pointing at the trigger. When it starts open (`default-open`) it
does not take focus, so a page that hydrates keeps its own. Modal, the page
behind is inert. It fades out when it closes, unless motion is reduced.

`placement` is a floating-ui placement (`bottom`, `top-start`, …); `start`
and `end` follow the text direction.
