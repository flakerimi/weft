---
group: basics
purpose: Avatars overlapping in a row, showing the first few and counting the rest.
use: For the people on a project, a thread or a document, where the faces matter more than the names.
avoid: For a list a person reads name by name, use a list with an Avatar beside each name.
adapts:
  input: Nothing changes; the group is not a hit target.
  surface: Nothing changes.
  width: Nothing changes; set max so the row fits the narrowest place it appears.
---
Put Avatars in its default slot. With `max`, the first `max` show and a last
avatar counts the rest (`+3`); without it every avatar shows. `size` sizes
every avatar in the group.

It is a `group`: name it with an `aria-label` ("Project members"). Avatars
join the group in the order they render, on the server and in the browser
alike, so the count hydrates as rendered.
