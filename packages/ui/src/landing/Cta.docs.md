---
group: landing
purpose: A closing call-to-action band or card prompting the next step.
use: Near the foot of a landing or marketing page to prompt the visitor to install, start a trial or get in touch.
avoid: For the top of a landing page, use Hero; for a short notice across the top of a page, use Banner; for an empty list, use EmptyState.
adapts:
  input: Action buttons use size lg, at least --weft-hit-min (44 px) tall on pointer and touch alike.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with the landing type scale; in an app it runs full width.
  width: Measured from the block itself; a horizontal CTA puts its copy and actions (or aside) side by side when regular or wide, and stacks them when narrow (under 640 px).
---
`title` renders as an `h2` by default (`level` changes it to 2–6 to fit the
page's outline), with `headline` above and `description` below. `actions`
are `Button`s (`{ label, onClick, to, icon, tone, variant }`), the first
solid and the rest outlined; the `actions` slot replaces them (for a copyable
install command or newsletter form).

`variant` is `card` (a bordered raised panel, the default), `soft` (tinted in
`tone`), `solid` (filled in `tone`, where buttons default to the `neutral`
tone for contrast) or `plain` (unboxed). `orientation` is `vertical` (the
fixed default: copy and actions stacked, centred unless `align="start"`) or
`horizontal` (copy at the start, actions or the `aside` slot at the end when
regular or wide, stacked when narrow).
