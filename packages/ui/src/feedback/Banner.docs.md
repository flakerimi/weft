---
group: feedback
purpose: A strip across the top of a page or app for news that concerns everyone, which can be closed for good.
use: For maintenance windows, a new feature, a policy change: above the Header, on every page, until read.
avoid: For a message about one part of a page, use Alert; for a confirmation of something the person did, use a Toast.
adapts:
  input: On touch the close button's hit area is at least --weft-hit-min, and the actions are Buttons (their rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: When narrow the actions wrap under the title, both centred.
---
`title` (or the default slot) is the news, `icon` (or the leading slot)
before it, `actions` (`{ label, onClick, to }`, or the actions slot) after
it. `to` makes the title a link. `tone` (`primary`) fills the strip.

`close` adds a close button, which hides the banner and emits `close`;
focus, which was on that button, moves on to what follows the banner (else
what precedes it). With an `id` the banner stays closed on later visits: it
records `closed` under `weft-banner:<id>` in localStorage. The server always
draws the banner; a pre-paint snippet (`usePrePaint`, written into the
`<head>` only of pages that render a closable Banner with an id) hides a
closed one before the first paint, and the Banner leaves the page once
mounted. Change the id for new news. Without an id it shows again on the
next page.

`v-model:open` binds whether it shows: closing it sets `false` (as does a
closed `id` found once mounted), and the page can close or reopen it.

It is a plain block, not a landmark or a live region: put it before the
page's header so it is read first.
