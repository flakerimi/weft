---
group: forms
purpose: A number typed or stepped up and down, kept between a minimum and a maximum.
use: For a quantity, a count or an amount where a step up or down is natural.
avoid: For a value picked along a range, use Slider; for numbers that are really codes (a phone number, a card), use Input.
adapts:
  input: md is 44 px tall on touch, as an md Button is, and the steppers become large buttons, 44 px wide (28 px on pointer) and at least 44 px tall through their hit areas.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container.
---
Bind the number with `v-model`; it is `null` when the field is empty.

**Keyboard** (the WAI-ARIA spinbutton pattern): Up and Down step by `step`,
Page Up and Page Down by ten steps, Home and End go to `min` and `max`.
Typed text is read when it is committed - Enter, or leaving the field - and
kept within `min` and `max`; text that is not a number goes back to the
value. Stepping by decimals leaves no floating-point dust (0.1 + 0.2 is 0.3).

**Mouse wheel:** off by default, so scrolling the page never changes a value.
With `wheel`, the wheel steps it, but only while it has focus.

**Steppers:** out of the tab order (the keys do their work), named by
`decrement-label` and `increment-label` ("Decrease", "Increase"), disabled at
the bounds, and a press keeps focus where it was. The input is a text input
with `role="spinbutton"` and `inputmode="decimal"`, carrying
`aria-valuenow`, `aria-valuemin` and `aria-valuemax`.
