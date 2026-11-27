---
group: feedback
purpose: What a list or page shows when loading failed, with the error and retry button.
use: In place of a Table, a chart or a page whose request failed; it reads an Error, a string or the wire format's error body.
avoid: For a form field's error, use FormField; for a failure the person can fix in the page, use Alert; for nothing to show, use EmptyState.
adapts:
  input: On touch its buttons are at least --weft-hit-min tall (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: As EmptyState: the actions wrap and stay centred when narrow.
---
An EmptyState in the danger tone with `role="alert"`, so it is announced
when it appears. `title` defaults to "Something went wrong"; `description`
defaults to the message of `error`, which may be an `Error`, a string, or
the wire format's `{ error: { code, message } }` (its `code` shows small
under the message). A warning sign sits above; `icon` replaces it.

`retry` adds a button (`retry-label`, "Try again") that emits `retry`;
while `loading` (retrying) it spins, says `aria-busy` and takes no presses.
`actions` add more buttons after it; the `actions` slot replaces both with
buttons of your own. The default slot adds content under the message and
`footer` a support link. `level`, `variant` and `size` are
EmptyState's.
