---
group: data
purpose: A dialog that creates or edits a record from a Weft field schema.
use: For making or editing a resource record from a generated `schema.<Name>Input`, with client validation, diffed PATCH updates and server error mapping.
avoid: For a confirmation with no fields, use ConfirmDialog; for a dialog whose content is not an entity form, use Dialog with Form.
adapts:
  input: Inherits Dialog's and the inner form controls' touch hit areas (at least 44 px).
  surface: Nothing changes in behaviour; text and spacing follow the surface's type scale.
  width: Full-screen when the viewport is narrow (under 640 px), and enum fields with four or fewer choices switch from a horizontal RadioGroup to a Select.
---
It has no stylesheet of its own: it composes `Dialog`, `Form`, `Stack`,
`Alert`, `FormField`, `Button` and the form controls.

`EntityDialog` pairs `Dialog` and `Form` around a generated `schema`
(`schema.ProductInput` from `weft gen`). In create mode (`record` null or
omitted) it starts each field from `defaults` (or the schema's empty value)
and calls `create(body)` on submit. In edit mode (`record` set) it populates
the form from `record`, diffs the submitted values against `record`, calls
`update(record[idKey], changed)` with only the fields that changed, and closes
without a request when nothing changed.

**Controls from the schema.** Each field in `schema` renders a `FormField`
whose label defaults to the sentence-cased field name (`firstName` →
`First name`, `unit_price` → `Unit price`) and whose control follows the rule:
- `enum` (or `fields[name].items`) with four or fewer choices renders a
  horizontal `RadioGroup` on regular and wide viewports and a `Select` when
  narrow; five or more choices render a `Select`. An optional non-nullable
  enum prepends `labels.noneOption` (default `"None"`).
- `kind: 'boolean'` (or `type: 'boolean'`) renders a `Checkbox` (or a `Switch` with
  `fields[name].control = 'switch'`).
- `kind: 'number'` (or `type: 'number'`) renders an `InputNumber` with `min` and `max` from the
  rule and `step` from `fields[name].step` when given.
- `email: true` renders `<Input type="email">`, `url: true` renders
  `<Input type="url">`, `fields[name].control = 'textarea'` renders
  `Textarea`, and everything else renders `<Input type="text">`.

**Customizing fields.** `fields` configures any field by name (`label`,
`hint`, `description`, `placeholder`, `control`, `step`, `rows`, `items`,
`hidden`). The `#<name>-field="{ state, rule, config }"` slot replaces one
field's control inside its `FormField`, and the default slot
`#default="{ state, record, schema }"` replaces the whole field list while
keeping the top `Alert` and the Cancel and Create/Save buttons.

**Errors.** Before calling `create` or `update`, `Form` checks every field
against `schema` (with non-nullable number fields marked required so an
emptied number is caught before sending). When `create` or `update` rejects
with Warp's `ApiError` shape (`{ status, code, message, fields }`), field
errors matching rendered fields attach to their `FormField`s, unknown field
keys or a top-level `message` appear in an `Alert` at the top of the form,
`code === 'network'` shows `labels.unreachable`, and any other error logs to
`console.error` and shows `labels.failed`. Reopening the dialog or switching
`record` resets the draft and clears every error.
