---
group: dates
purpose: A month of dates to choose one, several or a range from, in the locale's names, digits and first day of the week.
use: For a date chosen in view of the month around it (a booking, a schedule, a range of nights), shown inline on the page or inside a DatePicker.
avoid: For a date people know and type (a birthday), use DatePicker, whose field takes typing; for a time of day, InputTime.
adapts:
  input: On pointer a date is the size's square (36 px at md). On touch every date and page button is at least 44 px a side, the dates share the width the calendar has, and a swipe across the dates changes the month (leftwards the next one; mirrored right to left); a mouse never swipes.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It is as wide as its months; with number-of-months above 1 the months wrap onto new lines when the container is narrower than them.
---
Bind the date with `v-model`: an ISO string (`2026-11-21`) by default, or a
`CalendarDate` of `@internationalized/date` with `value-format="object"`. With
`multiple` it binds a sorted array, a press toggling a date; with `range` it
binds `{ start, end }` once both ends are chosen (a first press starts the
range, the pointer or the focus previews it, a second press ends it, in
either order; a range cannot span a disabled date, so a press past one
starts again from it). A single date is chosen as a radio is: pressing it
again keeps it. A press on a day of the next or previous month shows that
month, focus going with it (whether or not the browser focused the button
it pressed: Safari and Firefox on macOS do not); a script choosing one leaves
focus where it is.

`type="month"` chooses a month instead (bound as its first day,
`2026-03-01`) from a grid of twelve, the heading switching to the years and
back; `type="year"` chooses a year (its 1 January) from a page of twelve. Both
take `multiple` and `range` (a range from the first day of its first month
or year to the last day of its last: `{ start: '2026-03-01', end:
'2026-05-31' }`).

**First render.** With no value and no `focused-date`, the month to show is
today's, which only the browser knows (a page built last week, or served
from another time zone, would draw another month and hydrate with a
mismatch). So until it is mounted it draws an empty grid, `aria-busy="true"`,
its cells the size of dates and its weekday headers in place, the same on
the server and while hydrating; then today's month. With `type="month"` or
`"year"` the empty grid is twelve blank cells, then this year's months, or
the page of years holding it. Give it a value or a
`focused-date` to draw dates on the server. Every computation goes through `@internationalized/date`;
dates are Gregorian.

It follows the grid of the WAI-ARIA APG Date Picker Dialog: each month is a
`table` with `role="grid"`, labelled by the heading (`aria-live="polite"`, so
paging is announced) or, with several months, by the month's own label;
weekday headers are `th scope="col"` with the full name in `abbr`; each date
is a button in a `gridcell` carrying `aria-selected`, named in full
("Saturday, November 21, 2026"), today marked `aria-current="date"` (once
mounted: the server and the browser may not agree on the day). One date is
the tab stop, the focused date, bound with `v-model:focused-date`; it decides
the month shown and is marked `data-weft-autofocus`, so a Popover, Dialog or
Sheet opening around the calendar focuses it first.

**Keyboard.** Left and Right move a day (mirrored right to left), Up and Down
a week, Home and End to the first and last day of the week as the locale
counts it, Page Up and Page Down a month (the day kept, or the month's last),
Shift+Page Up and Shift+Page Down a year; Enter and Space (the button's own
press) choose. Moving past the month shows the next one. The heading is a
button switching to a month view, then a year view (twelve years), keyed the
same way (a row is three months or years; Page Up and Page Down a year or
twelve); choosing a month or year goes back down (with `type`, down to the
unit chosen, never below it). The page buttons move a
month and a year (a year and twelve years in the other views), and keep focus.

`min` and `max` bound what can be reached: the keyboard stops there, the page
buttons are disabled past them, and the dates beyond are `aria-disabled`.
`is-date-disabled` dates can be reached but not chosen; `is-date-unavailable`
ones are drawn struck through and cannot be chosen either. `readonly` lets
the dates be focused, not chosen; `disabled` turns every button off. Names,
digits and the first day of the week follow `locale` (else the WeftProvider's,
else `en-US`, never the browser's, so server and browser agree);
`week-starts-on` (0 Sunday to 6 Saturday) overrides the first day, and
`weekday-format` the headers. `number-of-months` shows months side by side
(the days of the next and previous month left blank); `fixed-weeks` (on by
default) keeps six weeks; `week-numbers` adds ISO 8601 week numbers. `labels`
translates the page buttons' names. Slots: `day`, `heading`, `week-day`.
