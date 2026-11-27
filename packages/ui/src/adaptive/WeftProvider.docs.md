---
group: adaptive
purpose: Sets the input, surface and width context for every Weft component inside it.
use: Around a layout to declare its surface (landing for marketing pages, app for tools), and to force an input for a kiosk, a preview or a test.
avoid: For choosing a component's look, use its own props; never use it to make a mobile version of something, since components adapt themselves.
adapts:
  input: Left unset, it passes on the browser's detected input, pointer or touch, live (pointer until mounted, so a phone hydrates what the server drew); set, it overrides it for everything inside.
  surface: Left unset, it passes on the outer one (app by default); set, it switches density, type scale and spacing for everything inside.
  width: Nothing changes once mounted, since width comes from each component's own container; set, it is the width class drawn before anything is measured, on the server and while hydrating.
---
Its element carries `data-weft-surface`, and `data-weft-input` when its
`input` is set, which the theme's CSS reads to switch control heights, hit
sizes and density. With no input set the theme follows `(pointer: coarse)`
at the root, so a page that is never hydrated still has 44 px targets on a
phone. Components read the detected input as `pointer` until they have
mounted, then switch live: the server cannot know it, and a phone hydrates
the pointer markup first. It renders
with `display: contents`, so it does not affect layout. Nested providers
override one axis and inherit the other.

`locale` (a BCP 47 tag, `de-DE`) sets the language of dates, times and sizes
for every component inside that has no `locale` of its own (Calendar,
DatePicker, DateRangePicker, InputTime, FileUpload): names, digits, the order
of day, month and year, the first day of the week. Unset, it passes on the
outer one; with none at all they use `en-US`, never the browser's language,
so the server renders what the browser hydrates.

`inset` says everything inside is a preview of a page, drawn inside another
(a gallery's stage or thumbnail, a screenshot frame): a Page or an AppShell
in it renders no `<main>` landmark, and a Page's title is an `h2`, so the
page around keeps one main and one h1. An AppShell in it keeps no collapsed
state in localStorage. An inset provider makes every one inside it inset too.

`width` (`narrow`, `regular` or `wide`) is the width class everything
inside is drawn with before it can be measured: on the server, and while
hydrating. Unset, it passes on the outer one, and with none at all
components start `wide`. Once mounted, each measures its own box and
follows it. Give it to a part drawn at a known width that is never
hydrated, such as a picture of a phone, so a Table in it is drawn as its
cards.
