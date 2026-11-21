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

`close` adds a close button, which hides the banner and emits `close`. With
an `id` the banner stays closed on later visits: it records `closed` under
`weft-banner:<id>` in localStorage and reads it once mounted, so the server
always draws the banner and a closed one leaves just after the page loads.
Change the id for new news. Without an id it shows again on the next page.

It is a plain block, not a landmark or a live region: put it before the
page's header so it is read first.
