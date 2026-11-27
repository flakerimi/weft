---
group: dates
purpose: A time of day, typed part by part in the locale's order and hour cycle.
use: For a time in a form - an opening hour, a meeting's start, an alarm - bound as HH:mm.
avoid: For a duration, use InputNumber; for a date, DatePicker.
adapts:
  input: On pointer it is a group of spinbuttons, one per part of the time, each a tab stop, keyed and typed into. On touch it is the platform's time field, whose own picker the phone draws (a wheel or a clock), with the same min, max and step, 44 px tall; it shows the time in the phone's own format. The input can change while it has focus; the time stays and focus moves to the other form's first part.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container, as Input does.
---
Bind the time with `v-model`: `HH:mm` (`HH:mm:ss` with `granularity="second"`,
`HH:00` with `hour`), or a `Time` of `@internationalized/date` with
`value-format="object"`; it is null until every part is filled. With a `name`
it submits that string in a hidden input. Outside a FormField, with no `id`
prop, it takes a generated id (on touch, the native time input's; a range's
two are that id with `-start` and `-end`).

On pointer the parts follow the locale (`1:05 PM` in en-US, `13:05` in en-GB,
`오후 1:05` in ko-KR); `hour-cycle` (12 or 24) overrides its hour cycle. Each
part is a `spinbutton` with `aria-valuenow`, `aria-valuemin`,
`aria-valuemax` and `aria-valuetext`, named "Hour", "Minute", "Second" and
"AM/PM" (`labels` translates them); the group has `role="group"`, named by
its FormField's label (`aria-labelledby`, the label a `<span>` rather than a
`<label>`, since a group is no form control) and described by its hint and
error.

**Keyboard.** Up and Down step the part (by `step`, wrapping round), Home and
End go to its ends, digits type into it (in the locale's digits too; the
next part takes focus once this one is full, or once no further digit could
follow), A and P (or the locale's words' first letters) set AM and PM,
Backspace clears a digit, then the part, Left and Right move between parts
(mirrored right to left), and a separator typed moves on.

With `range` it is two times, a start and an end, bound as `{ start, end }`
(each null while empty; null when both are) and submitted as `name[start]`
and `name[end]`: on pointer two groups of parts, named "Start time" and "End
time", Left and Right going on from one to the other; on touch two platform
fields so named. An end before the start is marked invalid.

A time before `min` or after `max` is marked invalid (`aria-invalid`,
`data-invalid`) but still bound, as a native time field does. `disabled`
takes the parts out of the tab order; `readonly` keeps them focusable and
unchanged. The `leading` slot (or `icon`) draws the box's start.
