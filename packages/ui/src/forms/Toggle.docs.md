---
group: forms
purpose: A button that stays pressed or not, such as Bold in a text toolbar.
use: For an on-off choice shown as a button among other buttons (bold, mute, pin), where the press itself says the state.
avoid: For a setting with a label beside it, use Switch; for one choice among several buttons, use ToggleGroup; for an action, use Button.
adapts:
  input: On touch an md toggle is at least --weft-hit-min (44 px) tall, a small or square one through its hit area (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; it does not wrap.
---
Bind `v-model` to a boolean. It is a Button that says `aria-pressed`, so
assistive technology reads it as a toggle button (APG button pattern); a
press, Enter or Space changes it. Not pressed it is drawn with `variant`
and `tone` (`ghost`, `neutral`), pressed with `active-variant` and
`active-tone` (`soft`, `primary`). `label`, `icon`, `trailing-icon`,
`square`, `size` and the leading, default and trailing slots are Button's;
with only an icon give it an `aria-label`.

With a `name` it posts `value` (default `on`) with a plain form while
pressed, and nothing while not, as a checkbox does. `loading` (the change is
being saved) shows a spinner, says `aria-busy` and blocks every press;
`disabled` is the native attribute. In a FormField it takes the field's id
and disabled state.

It has no stylesheet of its own: it is drawn by Button's.
