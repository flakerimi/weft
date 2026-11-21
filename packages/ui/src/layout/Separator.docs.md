---
group: layout
purpose: A rule that separates content, with an optional label or icon in its middle.
use: Between groups of content that need a named break, such as "or" between two sign-in methods, or a date in a feed.
avoid: For a plain rule, use Divider; for space alone, use the gap of a Stack or Grid; for a new titled part, use Section.
adapts:
  input: Nothing changes.
  surface: Nothing changes; the label follows the surface's type scale.
  width: Nothing changes.
---
`label` (or the default slot) and `icon` sit between two lines. It is a
`separator` with `aria-orientation`, named by `label`; `decorative` hides it
from assistive technology (`role="none"`, `aria-hidden="true"`), for a rule
that separates nothing a reader needs told about. `orientation="vertical"` stretches to the height
of its row.
