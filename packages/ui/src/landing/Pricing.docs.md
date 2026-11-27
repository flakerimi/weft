---
group: landing
purpose: A pricing section of plan cards, with an optional billing-cycle toggle.
use: On a landing or upgrade page to compare plans and let a visitor switch between billing cycles such as monthly and yearly.
avoid: For a full matrix of dozens of rows across plans, pair Pricing with a Table below it; for a single product card, use Card.
adapts:
  input: The ToggleGroup cycle buttons and each plan's call-to-action Button are at least --weft-hit-min (44 px) tall on touch.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with the landing type scale; in an app it runs full width.
  width: The inner Grid adapts in CSS first from its own container width (@container), stacking plans in 1 column when narrow, up to 2 when regular, and up to the plan count (at most 4) when wide.
---
`plans` are `{ id, title, description, price, period, billingNote, badge,
highlight, features, action, slot }`. The section's `title` renders as an
`h2` by default (`level` changes it to 2–6), and each plan's `title` renders
one heading level deeper (`h3` under `h2`). A plan with `highlight: true`
draws an accent border in `tone` and defaults its `action` button to `solid`
(non-highlighted plans default to `outline`).

**Billing cycles.** When `cycles` (`{ label, value, badge }[]`) is passed,
`Pricing` renders a single `ToggleGroup` above the plans, bound with
`v-model:cycle` (or `defaultCycle`, defaulting to the first cycle's `value`),
keeping the active cycle selected if pressed again. Switching cycles runs in
the browser, so a `Pricing` with `cycles` needs an `<Island>` around it; a
static `Pricing` with no `cycles` needs no JavaScript. Each plan's `price`,
`period` and `billingNote` can be a plain string (the same for every cycle,
such as `price: '$0'`) or a map keyed by cycle `value` (`price: { monthly:
'$29', yearly: '$24' }`). A `price` or `period` map that lacks the selected
cycle leaves that plan's price area empty (never showing another cycle's
price) and warns once in development naming the plan and the cycle; a
`billingNote` map that lacks the selected cycle shows no billing note for
that cycle.
