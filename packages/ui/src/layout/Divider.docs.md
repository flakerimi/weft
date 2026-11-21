---
group: layout
purpose: A thin rule that separates content, across or alongside it.
use: Between groups of content that need a visible break, in a menu, a card or a toolbar.
avoid: For a rule with a label or icon, use Separator; for space alone, use the gap of a Stack or Grid; for a new titled part, use Section.
adapts:
  input: Nothing changes.
  surface: Nothing changes.
  width: Nothing changes.
---
Horizontal is an `hr`; vertical is an element with `role="separator"` and
`aria-orientation="vertical"`, which stretches to the height of its row.
