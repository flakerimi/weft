---
group: disclosure
purpose: Content shown and hidden by one trigger.
use: For detail most readers skip (advanced settings, a long description, the rest of a list).
avoid: For several sections of which one or a few are open, use Accordion; for content that floats over the page, use Popover.
adapts:
  input: The drawn trigger is at least --weft-hit-min tall on touch; a slotted trigger (a Button) brings its own touch size.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The content takes the width of the collapsible at every width.
---
Bind the state with `v-model:open` (or start it with `default-open`). The
default slot is the trigger, one element (a Button): it gets
`aria-expanded` and `aria-controls` once mounted, and a press on it toggles
the content. Without a default slot, `label` draws a trigger button with a
chevron, which states both from the server on. The `content` slot is what
opens.

It follows the WAI-ARIA APG Disclosure pattern. **Keyboard:** the trigger is
a button, so Enter and Space toggle it.

**Find in page.** The content is always rendered, on the server too, and
hidden with `hidden="until-found"`: the browser's find-in-page searches it,
and when it finds text there the collapsible opens (`beforematch`). A
browser without `until-found` treats it as plain `hidden`.

**Motion.** The height animates between 0 and its content in CSS
(`interpolate-size`); under reduced motion it opens at once, as it does in a
browser without `interpolate-size`. `disabled` keeps it as it is.
