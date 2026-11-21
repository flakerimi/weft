---
group: overlays
purpose: The inside of a modal panel: a header with the heading, actions and close button, a body that scrolls, and a footer.
use: Inside a modal surface of your own that should look like Dialog, Sheet and Drawer, which render it themselves.
avoid: To open a dialog, use Dialog, Sheet or Drawer; for a card on the page, Card.
adapts:
  input: Its close button's hit area is at least 44 px on touch.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; the body scrolls when the panel is short.
---
`title` and `description` are the heading; the `header` slot replaces both.
`title-id` goes on whatever heading renders (the `<h2>`, or the element
around the header slot), and `description-id` on the description, so the
overlay's `aria-labelledby` and `aria-describedby` always name something.
`close` shows a CloseButton; it, and `close()` in the `body` and `footer`
slots, emit `close`. The `actions` slot sits before the close button.
