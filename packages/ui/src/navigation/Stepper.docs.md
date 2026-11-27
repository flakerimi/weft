---
group: navigation
purpose: The steps of a task done in order, with the current step's content.
use: For a checkout, an onboarding, a setup wizard: three to six steps.
avoid: For views a reader moves between freely, use Tabs; for where a page sits in a site, Breadcrumb.
adapts:
  input: Each step button is at least --weft-hit-min tall on touch.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: A horizontal stepper stacks vertically when its own width is narrow (under 640 px). CSS does it from `data-width`, which the page's pre-paint script sets, so a page that is never hydrated adapts too; the arrow keys follow once hydrated.
---
`items` are `{ title, description, value, icon, content, disabled, slot }`;
a step's value is its `value`, else its index. Bind the current step with
`v-model`. The `indicator`, `title` and `description` slots replace those
parts of a step, the `content` slot the current step's content, and a slot
named by an item's `slot` renders that step's content.

The steps are an ordered list of buttons; the current one says
`aria-current="step"`, those before it are drawn done (a check), those after
it waiting. `linear` (the default) lets a press reach only the steps done and
the next one; disabled steps are never reached. The component exposes
`next()`, `prev()`, `hasNext` and `hasPrev` (through a template ref), for a
form's Back and Continue buttons, and emits `next` and `prev`.

**Keyboard.** The step buttons are one tab stop: the arrow keys move
between them (Left and Right mirrored right to left), Home and End go to the
ends, Enter and Space choose.
