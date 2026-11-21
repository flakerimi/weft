---
group: disclosure
purpose: Stacked sections whose content opens under its header.
use: For questions and answers, settings in groups, any long page whose sections a reader opens one at a time.
avoid: For one show and hide, use Collapsible; for views of the same thing that replace each other, use Tabs.
adapts:
  input: Each header is a full-width button, at least --weft-hit-min tall on touch.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills its container; headers wrap their text when narrow.
---
`items` are `{ label, value, icon, trailingIcon, content, disabled, slot }`.
A section's value is its `value`, else its index as a string. `type`
`single` (the default) keeps one section open, bound with `v-model` as its
value (`null` when none is); `multiple` binds an array of values. With
`single`, `collapsible` (the default) lets the open section close again;
false keeps one open always. `disabled` stops every section. The `label`
slot replaces a header's text, the `content` slot a panel's content, and a
slot named by an item's `slot` renders that item's panel.

It follows the WAI-ARIA APG Accordion pattern. Each header is a heading
(`h3`; `level` changes it to fit the page's outline) holding a button with
`aria-expanded` and `aria-controls`; each panel is a `region` named by its
header. When the open section of a non-collapsible accordion cannot close,
its header says `aria-disabled="true"`.

**Keyboard.** Enter and Space toggle the focused section. ArrowDown and
ArrowUp move to the next and previous header, wrapping, Home and End to the
first and last; disabled headers are skipped. Every header stays in the tab
order.

**Find in page.** Panels are rendered, on the server too, and hidden with
`hidden="until-found"`: find-in-page searches them and opens the section it
finds text in. Heights animate in CSS, not under reduced motion.
