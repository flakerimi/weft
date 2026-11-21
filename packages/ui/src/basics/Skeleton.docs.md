---
group: basics
purpose: A placeholder in the shape of content that is still loading.
use: Where text, an avatar or a card will appear, so the layout does not jump when it arrives.
avoid: For a wait with no shape to show, use Spinner; when the progress is known, use Progress.
adapts:
  input: Nothing changes.
  surface: Nothing changes; its default height is one line of the surface's text.
  width: Nothing changes; without a width it fills its container.
---
`width` and `height` take any CSS length; without them it is the full width
and one line (1em) high. `circle` makes it round, for an avatar.

It pulses, and stands still under `prefers-reduced-motion`. It is hidden
from assistive technology: set `aria-busy="true"` on the region that is
loading, and remove it when the content arrives.
