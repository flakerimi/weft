---
group: basics
purpose: A short label for a status, a category or a count, drawn in a tone.
use: Beside a title or in a table cell, to say a state at a glance (Paid, Failed, Beta, 3).
avoid: For something a person can press or remove, use Button; for an indicator on the corner of an icon or avatar, use Chip.
adapts:
  input: Nothing changes; a badge is not a hit target.
  surface: Nothing changes; text follows the surface's type scale.
  width: Nothing changes; a badge does not wrap.
---
`variant` is `solid`, `soft`, `outline` or `ghost`; `tone` is `neutral`,
`primary`, `danger`, `warning` or `success`; `size` is `sm`, `md` or `lg`.
Give the text as `label` or as the default slot. Icons are decorative, so the
text must carry the meaning; with only an icon the badge is square and needs
an `aria-label`.
