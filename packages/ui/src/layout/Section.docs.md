---
group: layout
purpose: A titled part of a page, with an optional description and actions.
use: To split a page into named parts, each with an h2, such as Billing and Members on a settings page.
avoid: For the page itself, use Page; for a bordered box, use Card.
adapts:
  input: Nothing changes.
  surface: Nothing changes; the space between sections comes from Page.
  width: Nothing changes; the actions wrap under the title when they do not fit beside it.
---
The `title` renders as an `h2`.
