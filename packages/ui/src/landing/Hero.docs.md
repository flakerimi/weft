---
group: landing
purpose: The opening band of a landing page, with a title, lead, actions and visual.
use: At the top of a product's or documentation site's home page to state what it is and offer the main action.
avoid: For the top of an app page inside an AppShell, use Page; for a closing call to action lower on the page, use Cta.
adapts:
  input: Action buttons use size lg, at least --weft-hit-min (44 px) tall on pointer and touch alike.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with the larger landing type scale; in an app it runs full width.
  width: Measured from the hero's own box (useWidth, before paint and on resize); a horizontal hero puts the copy and visual side by side when wide (1024 px and up) and stacks the copy above the visual when regular or narrow.
---
`title` renders as the page's `h1` by default, or as an `h2` inside a
`WeftProvider` marked `inset` (so a preview stage never adds a second `h1` to
the page around it); `level` overrides it. `headline` (or the `headline`
slot) sits above the title, `description` below it, and `actions` renders
`Button`s (`{ label, onClick, to, icon, tone, variant }`), the first solid
primary and the rest outlined; the `actions` slot replaces them (for an
install command or prompt box) and `caption` adds a quiet note below.

`orientation` is `vertical` (the fixed default: copy above, visual below,
centred unless `align="start"`) or `horizontal` (copy at the inline start,
visual in the default slot at the inline end when wide, stacked when narrow
or regular). The `background` slot places a decorative layer behind the
container (`aria-hidden="true"`).
