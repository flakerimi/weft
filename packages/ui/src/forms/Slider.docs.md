---
group: forms
purpose: A value, or a range between two values, picked along a track.
use: For a value where the position matters more than the exact number (volume, a price range).
avoid: For an exact number, use InputNumber; for a few named steps, use RadioGroup.
adapts:
  input: The thumb is drawn 16 px (12 sm, 20 lg) on pointer and 24 px on touch (--weft-thumb-min), where a finger covers it; on touch its hit area and the track's grow to 44 px through pseudo-elements, and the slider is 44 px tall.
  surface: Nothing changes.
  width: It fills the width of its container; vertical sliders are 160 px tall.
---
Bind a number with `v-model`, or an array for a range: a thumb is drawn per
value, thumbs never pass each other (`min-steps-between-thumbs` keeps a gap),
and the track between them fills.

**Keyboard** (the WAI-ARIA slider pattern): each thumb is a `role="slider"`
in the tab order. Right and Up raise it by `step`, Left and Down lower it,
Page Up and Page Down move ten steps, Home and End go to the ends. In a
right-to-left document Left and Right swap, and the track fills from the
right. Each thumb's `aria-valuemin` and `aria-valuemax` are its neighbours.

**Pointer:** a press on the track moves the nearest thumb there, on a step,
and drags it until let go. `update:modelValue` fires as it moves; `change`
fires once a move ends (a key press, or a drag let go).

Thumbs are named by `thumb-labels`, or for a range "Minimum" and "Maximum";
a single thumb takes the Slider's own `aria-label`, or its FormField's label.
In a FormField a range's thumbs are named by the label and their own end
("Price Minimum"). With `name`
each value posts from a hidden input.
