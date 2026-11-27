---
group: feedback
purpose: An inline message asking for attention, in a tone, with actions and close.
use: Above a form that failed, at the top of a page whose data is stale, beside a setting with a consequence (a trial ending, a payment declined).
avoid: For news about the whole site or app, use Banner; for a passing confirmation, use a Toast; for a page with nothing to show, use EmptyState or ErrorState.
adapts:
  input: On touch the close button's hit area is at least --weft-hit-min, and the actions are Buttons (their rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Horizontal actions wrap under the text when its box has no room for them beside it.
---
`title` is the message, `description` (or the default slot) says more,
`icon` (or the leading slot) sits before them. `tone` (`primary`) says what
it is about: `danger` for a failure, `warning`, `success`, `neutral`.
`variant` is `soft` (default, tinted), `solid` (filled), `outline` or
`ghost`.

`actions` are Buttons given as data, `{ label, onClick, to, icon, tone,
variant }`, or the actions slot; they sit under the text, or beside it with
`orientation="horizontal"`. `close` adds a close button (`close-label`,
"Close"); pressing it sets `v-model:open` to false and the alert leaves the
page, focus moving on to what follows it (else what precedes it). Closed by
its parent, it leaves focus where it is.

It has no live role of its own: a message that is there when the page
loads is not news, and announcing every alert would drown the important
ones. Give `role="alert"` to one that appears after something failed and
must be heard at once, or `role="status"` to a polite one; either must be
in the page before its text changes to be announced.
