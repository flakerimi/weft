---
group: layout
purpose: Centres content at the page width, with inline padding.
use: Around the content of a landing page band (a hero, a feature grid, a footer) whose background runs full width.
avoid: For an app page with a title and actions, use Page, which also sets the width; for grouping things, use Card or Section.
adapts:
  input: Nothing changes.
  surface: On landing its width is capped at the page width (--weft-page-max, 1200 px in the shipped themes); in an app it runs full width.
  width: Its inline padding grows with the room around it, from 16 px to 32 px, without a breakpoint.
---
`as` renders another element, such as `section`, `header` or `footer`.
