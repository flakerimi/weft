---
group: overlays
purpose: A modal window over the page, for a task or message that needs the user's attention before going back.
use: For a short form, a decision or details that interrupt the page on purpose: rename, invite, settings of one thing.
avoid: To confirm an action, use ConfirmDialog; for a panel at the edge (navigation, filters), Sheet; for a hint, Tooltip or Popover.
adapts:
  input: Nothing changes in behaviour; on touch its close button's hit area is at least 44 px.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Centred, at most 32rem wide; when the viewport is narrow (under 640 px) it is full-screen, as it is at every width with `fullscreen`.
---
The default slot is the trigger, one element (a Button), which opens it and
carries `aria-haspopup="dialog"`, `aria-expanded` and `aria-controls`; or
bind `v-model:open`. `title` names it and `description` describes it
(`aria-labelledby`, `aria-describedby`). The `body` and `footer` slots get
`close()`; `header` replaces the title and description, and `actions` adds
controls before the close button (×, drawn inline; `close="false"` hides it).

It follows the WAI-ARIA APG Dialog (Modal) pattern: `aria-modal="true"`,
focus trapped inside, starting on the first tabbable element that is not the
close button, and the page does not scroll while it is open.

**Closing.** Escape, a press on the backdrop and the close button close it;
`dismissible="false"` keeps it open on Escape and the backdrop. Focus returns
to its trigger, after a press on the backdrop too (the backdrop is the
dialog's own), or, with no trigger, to the element that had focus when it
opened.

Its content renders into `#weft-overlays` only once mounted, so the server
renders none of it; attributes go to the dialog element.
