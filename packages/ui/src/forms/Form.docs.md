---
group: forms
purpose: A form that checks its values against a Weft field schema before it submits.
use: For any form whose values need checking before they are sent, with the same rules the server enforces.
avoid: For a search box or a single setting that takes effect at once, a Form is not needed.
adapts:
  input: Nothing of its own; the fields inside it adapt.
  surface: Nothing changes in behaviour.
  width: Nothing changes; its fields stack in a column.
---
`state` is the values, an object the fields bind into with `v-model`;
`schema` is a Weft field schema - plain JSON a server can send, the same
shape the Go contract will generate - keyed by field name:

    { email: { required: true, email: true }, age: { min: 18, message: 'You must be 18 or over.' } }

Each rule is optional: `required`, `min` and `max` (a length for text and
lists, bounds for numbers), `pattern` (matched against the whole text, as the
HTML attribute does), `email`, and `message` in place of the default
sentences. Its TypeScript type is `FieldSchema`, and `validateFields` checks
values against one anywhere.

**When it checks.** On submit every field is checked. With errors, `error`
fires with them, each FormField with a matching `name` shows its error, and
focus moves to the first invalid control; without, the `@submit` handler
runs with a deep copy of the values (lists and objects too), typed as
`state` is. After the first submit, a field is checked again when it is
left, and a field showing an error is checked as it changes, so the error
goes the moment it is fixed - but no new error appears while typing.

**Server errors.** A template ref's `setErrors({ email: 'That address is
taken.' })` shows errors from elsewhere. Leaving such a field keeps its error
(the server said so, and nothing changed); changing the field checks it
again. `validate()` checks without submitting, and `clear()` clears every
error.

**Disabled and busy.** The content sits in a `<fieldset>` with no frame, so
`disabled` disables every control in the form at once - Weft's and native
ones, Buttons included - and a submit does nothing. `loading` does the same
and sets `aria-busy`. With `loading-auto` (on by default), a `@submit`
handler that returns a promise keeps the form busy until it settles, so a
second press or Enter submits nothing. (`@submit` is a handler prop, as in
Nuxt UI, so the form can wait for it.) The native form has `novalidate`:
Weft shows the errors, not the browser.
