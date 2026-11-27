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
`Alert`, `FormField`, `Button` and the form controls, and with
`useDateFields()` `DatePicker` and `InputTime`.

`EntityDialog` pairs `Dialog` and `Form` around a generated `schema`
(`schema.ProductInput` from `weft gen`). In create mode (`record` null or
omitted) it starts each field from `defaults` (an object or factory function,
or the schema's empty value) and calls `create(body)` on submit. In edit mode
(`record` set) it populates the form from `record`, diffs the submitted values
against `record`, calls `update(record[idKey], changed)` with only the fields
that changed, and closes without a request or `saved` event when nothing
changed.

**Controls from the schema.** Each field in `schema` renders a `FormField`
whose label defaults to the sentence-cased field name (`firstName` →
`First name`, `unit_price` → `Unit price`) and whose control follows the rule:
- `enum` (or a `fields` entry's `items`) with four or fewer choices renders a
  horizontal `RadioGroup` on regular and wide viewports and a `Select` when
  narrow; five or more choices render a `Select`. An optional non-nullable
  enum prepends `labels.noneOption` (default `"None"`).
- `kind: 'boolean'` (or `type: 'boolean'`) renders a `Checkbox` (or a `Switch`
  with `control: 'switch'`).
- `kind: 'number'` (or `type: 'number'`) renders an `InputNumber` with `min`
  and `max` from the rule and `step` from the field's `step` when given.
- `format: 'date'` (a Go field with the rule `date`) renders a `DatePicker`,
  and `format: 'datetime'` (any other `time.Time`) a `DatePicker` and an
  `InputTime` (hours and minutes) side by side in a `FieldGroup`, once
  `useDateFields()` is called in the setup of the island that draws the
  dialog (a `DataTable`'s too). Without it they are the browser's own
  `<Input type="date">` and `<Input type="datetime-local">`: a page's CSS is
  all its imports reach, so only a page that calls it pays for the date
  controls and the calendar. With them, the field's label is for the date,
  and the time is named `labels.timeOf(label)` (`"Starts, time"`). The draft
  holds what the wire holds: a date as midnight in UTC
  (`2026-03-29T00:00:00Z`), shown and entered as written in every time zone,
  and a datetime as an instant in UTC, shown and entered in the browser's
  zone (a date chosen before a time is at midnight there; a time the clocks
  skip, such as 02:30 when they go from 02:00 to 03:00, is moved on by the
  hour skipped, so it is saved as 03:30). Go's zero time is empty. Both start
  empty (`null`), an emptied one is `null`, and one not touched is not in
  the `update`, so a datetime's seconds survive an edit of other fields.
  `wireToDate`, `dateToWire`, `wireToLocal`, `localToWire` and
  `wireTimeText` (for a column's `format`) are exported for pages of your
  own.
- `email: true` renders `<Input type="email">`, `url: true` renders
  `<Input type="url">`, `control: 'textarea'` renders `Textarea`, and
  everything else scalar renders `<Input type="text">`. Non-scalar kinds
  (`list`, `record`, `object`) start as `[]` or `{}` and render only when
  given a `#<name>-field` slot.

**Customizing fields.** When `fields` is omitted, every field in `schema`
renders in key order. When `fields` is given, it takes an array of
`EntityFieldConfig` objects
(`{ name, label, hint, description, placeholder, control, step, rows, items, hidden }`),
rendering only the fields it lists, in that order, each merged with
`schema[f.name]`. The `#<name>-field="{ state, record, rule, field }"` slot
replaces one field's control inside its `FormField`, and the default slot
`#default="{ state, record, schema }"` replaces the whole field list while
keeping the top `Alert` (`role="alert"`) and the Cancel and Create/Save
buttons.

**Errors.** Before calling `create` or `update`, `Form` checks every rendered
field against `schema` (with non-nullable number fields in edit mode marked
required so an emptied number is caught before sending). When `create` or
`update` rejects with Warp's `ApiError` shape (`{ status, code, message, fields }`),
field errors matching rendered fields attach to their `FormField`s and focus
moves to the first invalid control once the form is no longer busy; unknown
field keys or a top-level `message` appear in an `Alert` (`role="alert"`) at
the top of the form; `code === 'network'` shows `labels.unreachable`;
`code === 'http'` shows `labels.unavailable`; and any other error logs to
`console.error` and shows `labels.pageFailed`. Reopening the dialog or
switching `record` resets the draft and clears every error.
