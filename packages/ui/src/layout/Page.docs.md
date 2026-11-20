---
group: layout
purpose: The main landmark of a page, with its title, description, actions and content.
use: Once per page, as the outermost layout component, so the page has one main, one h1 and the surface's width and spacing.
avoid: For a titled part inside a page, use Section; for a box of related content, use Card.
adapts:
  input: Nothing changes.
  surface: On landing the content is centred at up to 1200 px with wider gaps between sections; on app it fills the width with tighter gaps.
  width: Nothing changes in the component itself; the header's actions wrap under the title when they do not fit beside it.
---
The `title` renders as the page's `h1`. Give the `header` slot to replace the
title and description with your own markup, and the `actions` slot for
buttons beside them.
