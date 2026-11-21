---
group: overlays
purpose: A short text about a control, shown on hover or keyboard focus.
use: For naming an icon button, or adding a short hint to a control, where hovering is how a pointer user would ask.
avoid: For anything a touch user needs, put it in the interface, since tooltips do not show on touch; for interactive content, use Popover.
adapts:
  input: On pointer it shows on hover after `delay` and at once on keyboard focus. On touch it renders nothing, and nothing hovers; a trigger with no name of its own is named by the text (aria-label), so the meaning stays.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It wraps at 20rem, and stays in view by flipping and sliding.
---
The default slot is the trigger, one element; `text` is what it says (the
`content` slot replaces it). It follows the WAI-ARIA APG Tooltip pattern:
it never takes focus, and while shown the trigger is described by it
(`aria-describedby`, added after any description the trigger has).

It hides on Escape (only it, when it is the top overlay), on a press, when
the trigger loses focus, and when the pointer leaves both the trigger and the
tooltip, so the pointer can move onto it to read it. Once one tooltip has
hidden, the next shows without a delay for a moment, as when running the
pointer along a toolbar.

**Naming.** An icon-only trigger needs a name. When the trigger has none (no
`aria-label`, `aria-labelledby`, `title` or text), the tooltip's text becomes
its `aria-label`, on pointer and touch alike. A trigger that has a name keeps
it; on touch the tooltip's extra hint is then not shown, so do not put what a
person must know in a tooltip.

`v-model:open` shows it from code; `disabled` never shows it.
