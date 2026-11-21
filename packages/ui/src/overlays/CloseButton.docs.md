---
group: overlays
purpose: The × button that closes an overlay, with a touch-sized hit area.
use: In the header of a panel of your own, or a dismissible banner; Dialog, Sheet, Drawer and Toast draw it already.
avoid: For an action with a label (Cancel, Done), use Button.
adapts:
  input: It keeps its drawn size (28 px, 24 px small); its hit area is at least --weft-hit-min, so 44 px on touch.
  surface: Nothing changes.
  width: Nothing changes.
---
A native `<button type="button">` named by `label` (`Close`), drawn as a
cross inline, so a page with no CloseButton ships no icon for it. It emits
`click`. Focus shows the theme's ring.
