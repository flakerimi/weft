---
group: basics
purpose: A small indicator, a dot or a short count, on the corner of what it wraps.
use: For unread counts on an icon, online status on an avatar, a mark that something changed.
avoid: For a status in a line of text, use Badge.
adapts:
  input: Nothing changes; the indicator is not a hit target, the wrapped content is.
  surface: Nothing changes.
  width: Nothing changes.
---
Wrap the content in the default slot. `text` puts a short count or mark in
the indicator; without it the indicator is a dot. `position` is logical
(`top-end`, `bottom-end`, `top-start`, `bottom-start`), so `top-end` is the
top right in a left-to-right document and the top left in a right-to-left
one. `inset` keeps the indicator inside the corner, for round content.
`show` hides it, keeping the content.

Give `label` ("3 unread messages") to say what the indicator means: it is
read to assistive technology and the drawn indicator is hidden from it.
