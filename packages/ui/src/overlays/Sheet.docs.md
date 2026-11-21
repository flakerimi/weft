---
group: overlays
purpose: A modal panel that slides in from an edge of the viewport: a slideover from the side, or a bottom sheet.
use: For navigation on small screens, filters, a detail view or a form that should keep the page visible beside it.
avoid: For a short decision, use Dialog or ConfirmDialog; for content tied to one button on a large screen, Popover.
adapts:
  input: Nothing changes in behaviour; on touch its close button's hit area is at least 44 px. Popover and DropdownMenu render as a bottom sheet on touch with this stylesheet.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: A side sheet is at most 24rem wide, the full width when the viewport is narrower; a top or bottom sheet is the viewport's width and at most 85% of its height.
---
`side` is `start` or `end` (the inline sides, so mirrored in a right-to-left
document), `top` or `bottom`; the default is `end`. Otherwise it is used as
Dialog is: the default slot is the trigger, or bind `v-model:open`; `title`
and `description` name and describe it; `header`, `actions`, `body` and
`footer` slots, the last two with `close()`.

It is a modal dialog (WAI-ARIA APG, Dialog (Modal)): focus is trapped inside,
the page does not scroll, and Escape, the backdrop and the close button
close it (`dismissible="false"` keeps Escape and the backdrop from it), with
focus back on the trigger. The page behind is inert while it is open, and
it slides back out when it closes, unless motion is reduced; `overlay="false"`
draws no backdrop. The body scrolls when long.
