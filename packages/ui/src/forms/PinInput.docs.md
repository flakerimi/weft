---
group: forms
purpose: A short code typed one character per box.
use: For a one-time code, a verification code or a PIN.
avoid: For a password or anything longer than about eight characters, use Input.
adapts:
  input: md boxes are 44 px square on touch (36 px on pointer); with type number a number pad comes up.
  surface: Nothing changes.
  width: The boxes stay in one row.
---
Bind the joined code with `v-model`; `complete` fires once every box is
filled. `length` sets how many boxes (5). `type="number"` takes digits only;
`mask` hides what is typed; `otp` lets the browser fill a one-time code from
a message (`autocomplete="one-time-code"`).

**Keyboard:** typing a character fills the box and moves to the next;
Backspace clears the box, or steps back and clears the one before when it is
empty; Delete clears the box; Left and Right move (swapped in a right-to-left
document); Home and End go to the first and last box. A paste or an autofill
fills the boxes from the one it lands in, dropping characters the type does
not take.

The boxes form a `role="group"`, named by its FormField's label; each box is
named by its place (`box-label`, "Character 1 of 6"). With `name` the code
posts from a hidden input.
