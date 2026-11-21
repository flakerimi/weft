---
group: data
purpose: One figure that matters on a card, with how it changed and an optional sparkline.
use: In a row of key figures at the top of a dashboard (revenue, active customers, open tickets); it replaces the hand-built stat cards of admin panels such as Negenet's.
avoid: For many figures to compare, use Table; for a figure inside a sentence, write the sentence.
adapts:
  input: Nothing changes; it is not interactive (a link in its footer follows Link's rules).
  surface: Nothing changes in behaviour; the figure and label follow the surface's type scale.
  width: It fills its grid cell, and a long figure wraps rather than overflow; lay several out with Grid, which stacks them when narrow.
---
`label` names the figure and `value` is the figure: a number is formatted
for the locale (a WeftProvider's, else `en-US`, so the server and the
browser agree), text shows as given (`$1.2M`). It is a description list,
so assistive technology reads the label with the value.

`delta` is how it changed, as a number (4 for a 4% rise, -2 for a fall),
followed by `delta-unit` (`%`). Its sign picks the tone: success up, danger
down, neutral for 0; `invert` swaps them for figures where less is better
(churn, errors). It is drawn as `+4%` with an arrow and said as "up 4%",
"down 2%" or "no change" (translate with `labels`). `description` follows
the delta ("vs last month"). `icon` (or the leading slot) sits before the
label.

The `sparkline` slot holds a small chart under the figure, as wide as the
card, in the primary colour (an svg drawn with `currentColor` follows it);
it is drawing only, hidden from assistive technology, so the figure and the
delta must carry the meaning. `footer` holds a link to the details.
`loading` makes it `aria-busy`, a skeleton in place of the value and no
delta.
