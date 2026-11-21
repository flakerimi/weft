---
group: overlays
purpose: Asks the user to confirm an action before it goes ahead, with Cancel and a confirm button.
use: Before an action that is hard to undo: delete, discard changes, leave without saving.
avoid: For an action that can be undone, do it and offer Undo in a toast; for a form, use Dialog.
adapts:
  input: Nothing changes in behaviour; its buttons follow Button's touch sizes.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: As Dialog: full-screen when the viewport is narrow.
---
It has no stylesheet of its own: it is a Dialog with `role="alertdialog"`
(WAI-ARIA APG, Alert and Message Dialogs) and two Buttons.

`title` is the question, `description` what happens; `confirm-label` names
the action (`Delete`) and `tone="danger"` colours it for one that destroys.
Focus starts on Cancel, the safe answer.

`@confirm` runs on confirm and the dialog closes. When the handler returns a
promise, the confirm button loads and nothing closes the dialog until the
promise settles: it closes when the work succeeds and stays, ready again,
when it fails, so the person can retry or cancel, and emits `error` with
what the work failed with (with no `@error` listener it rethrows it, so it
is never swallowed). Cancel, Escape and the
backdrop emit `cancel` and close it. The default slot is the trigger, or bind
`v-model:open`; the `body` slot adds content under the description.
