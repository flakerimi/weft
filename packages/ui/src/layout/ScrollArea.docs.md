---
group: layout
purpose: A box whose content scrolls inside it, with the browser's own scrolling and thin themed scrollbars.
use: For a long list in a fixed-height panel, a wide strip of cards that scrolls sideways, a code block or a log.
avoid: For the page itself, let the page scroll; for thousands of rows, use Table or Listbox, which render only those in view.
adapts:
  input: On touch the browser's own overlay scrollbars and momentum scrolling apply; nothing is drawn over them.
  surface: Nothing changes.
  width: Nothing changes; it scrolls whatever overflows its box.
---
It is the browser's scrolling, not a drawn one: wheel, touch, momentum,
scroll anchoring, find-in-page and the keyboard all work as they do on the
page. The scrollbars are thin and drawn in the theme's border colour
(`scrollbar-width` and `scrollbar-color`, with the WebKit pseudo-elements
where those are missing).

`orientation` is `vertical` (default), `horizontal` or `both`. Give it a
height from outside, or `max-height`. While its content overflows it is a
tab stop (`tabindex="0"`), so a keyboard alone can scroll it with the
arrows, Page Up and Down, Home, End and Space; once it fits it leaves the
tab order. With `label` it is a named region, which assistive technology
lists; give one whenever it is a tab stop.

`shadow` fades the edges with more content beyond them (a mask, mirrored
right to left). Rows that must render only while in view belong in Table
or Listbox, which window long lists themselves.
