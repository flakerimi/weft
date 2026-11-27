---
group: landing
purpose: A titled FAQ section pairing a header with an until-found Accordion.
use: On a landing or pricing page to answer common questions without leaving every answer expanded at once.
avoid: When you only need the bare collapsible list without a section heading or side-by-side layout, use Accordion directly.
adapts:
  input: Each question header is a full-width Accordion button, at least --weft-hit-min (44 px) tall on touch.
  surface: On a landing surface its container is capped at --weft-page-max (1200 px) with its own block and inline padding (omitted inside a Page, Section or Container); in an app it runs full width.
  width: Measured from the block's own box (useWidth, before paint and on resize); a horizontal FAQ places the header and aside in a column beside the questions when wide (1024 px and up) and stacks them above when regular or narrow.
---
`items` uses `Accordion`'s item shape (`FaqItem = AccordionItem`: `{ label,
content, value, icon, trailingIcon, disabled, slot }`), so the same array
works in both and there is one way to write it. The section's `title` renders
as an `h2` by default (`level` changes it to 2–6), and each question header
in the `Accordion` renders one heading level deeper (`h3` under `h2`).

`orientation` is `vertical` (the default: header above the questions) or
`horizontal` (header and the `aside` slot in a column beside the questions
when wide, stacked above when narrow or regular). `type` (`single` or
`multiple`), `v-model`, `defaultValue` and `collapsible` work as on
`Accordion`, and every answer panel is rendered on the server too with
`hidden="until-found"` so browser find-in-page reveals a closed answer when
its text matches.
