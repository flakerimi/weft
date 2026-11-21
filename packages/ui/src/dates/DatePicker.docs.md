---
group: dates
purpose: A date field where a date is typed as the locale writes it or chosen from a calendar that opens beside it.
use: For one date in a form - a due date, a birthday, a start date - bound as an ISO string the server reads.
avoid: For a range of dates, use DateRangePicker; for a calendar always in view, Calendar; for a time of day, InputTime.
adapts:
  input: On pointer it is a text field, read in the locale on change and Enter, with a calendar button at its end that opens a non-modal floating calendar. On touch the whole field is one button showing the date, which opens the calendar in a modal bottom sheet titled with the field's label, its dates 44 px a side, a swipe changing the month. The input can change while the calendar is open; the same calendar stays, with the chosen date, the focused date and focus, and the button is the same element in both forms, so focus returns to it.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The field fills its container's width; the floating calendar stays within the room left, flipping above when there is none below; the sheet is the viewport's width.
---
Bind the date with `v-model`: an ISO string (`2026-11-21`), or a
`CalendarDate` with `value-format="object"`, null when empty. With a `name`
(or its FormField's) the ISO date submits in a hidden input; the text field
itself submits nothing.

**Typing.** The field shows the date as the locale writes it (`21/11/2026` in
en-GB, `11/21/2026` in en-US, `21.11.2026` in de; `format` takes other Intl
options), with the locale's pattern (`dd/mm/yyyy`) as its placeholder. On
change and on Enter it reads the text in the locale: its order of day, month
and year, any separator, its digits, a month name or abbreviation ("21 Nov
2026"), a two-digit year, no year (this year's), or ISO 8601. A date the
calendar allows is bound and written back the locale's way; empty text binds
null; anything else (no such date, before `min`, after `max`, disabled)
binds null, keeps the text and marks the field invalid (`aria-invalid`,
`data-invalid`), so it can be corrected.

**Calendar.** It follows the WAI-ARIA APG Date Picker Dialog: the button
(`aria-haspopup="dialog"`, `aria-expanded`, named "Choose date, 21 November
2026") opens it, as does Alt+ArrowDown in the field, on the date typed, else
the bound one, else today; focus goes to that date. It has the Calendar's
keys. Choosing a date binds it, closes the calendar (unless
`close-on-select` is false) and returns focus where it was opened from;
Escape closes it with focus back on the button; focus leaving it closes it
too. Its dialog is named by the field's label.

Inside a FormField the label is `for` the field (on touch, the button,
named by the label and the date), the description and error describe it, and
`required` marks it. `disabled` and `readonly` block typing and opening.
`min`, `max`, `is-date-disabled`, `is-date-unavailable`, `week-starts-on`,
`week-numbers` and `locale` go to the calendar and to reading typed dates;
`labels` translates the button's and the calendar's words. The `day` slot
draws the calendar's dates, `leading` the field's start.
