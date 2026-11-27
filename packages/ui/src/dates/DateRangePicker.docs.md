---
group: dates
purpose: A field for a start and end date range, typed in the locale or in a calendar.
use: For a stay, a report period or a leave request in a form, bound as two ISO dates the server reads.
avoid: For one date, use DatePicker; for a range always in view, Calendar with range.
adapts:
  input: On pointer it is two text fields, read in the locale, with a calendar button that opens two months side by side in a non-modal floating calendar. On touch the whole field is one button showing the range, which opens one month in a modal bottom sheet, dates 44 px a side, a swipe changing the month. The input can change while the calendar is open; the same calendar stays, with a half-chosen range, the focused date and focus.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The field fills its container's width; the floating calendar's two months wrap onto two lines when there is no room for both side by side; the sheet is the viewport's width.
---
Bind the range with `v-model`: `{ start, end }`, each an ISO string (or a
`CalendarDate` with `value-format="object"`) or null, and null when both are
empty. With a `name` the dates submit in hidden inputs as `name[start]` and
`name[end]`.

The two fields read typed dates as DatePicker's does, each on change and
Enter; an end before the start (or a start after the end) is refused, bound
as null and marked invalid with its text kept. Each field is named by the
FormField's label and its end ("Stay Start date"; without a FormField,
"Start date" and "End date", which `labels` translates).

The calendar is a Calendar with `range`: the first press (or Enter) starts a
range, the pointer or the focus previews it, the second ends it, binds it,
closes the calendar (unless `close-on-select` is false) and returns focus
where it was opened from. It opens on the start typed, else the end, else
today, two months on pointer and one on touch (`number-of-months` sets it).
Its button, keys, dialog, Escape, field wiring, `disabled`, `readonly`,
`min`, `max`, disabled dates and slots are DatePicker's.

`:portal="false"` draws the calendar's popover in place, as Popover's
`portal` does, so a page can show a picker open (`default-open`) with no
script: the calendar hangs below the field until mounted.
