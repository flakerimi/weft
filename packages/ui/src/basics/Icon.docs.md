---
group: basics
purpose: A Lucide icon by name, drawn as an inline SVG in the current text colour.
use: Wherever an icon helps recognition, beside a label or, with a label of its own, standing alone; Weft components take the same names in their icon prop.
avoid: For pictures and logos, use an img; never let an icon alone carry meaning without a label.
adapts:
  input: Nothing changes; an icon is not a hit target, so put it in a Button or Link for one.
  surface: Nothing changes; without a size it scales with the surrounding text.
  width: Nothing changes.
---
Names are Lucide's, as listed on lucide.dev/icons: `trash`, `chevron-down`.
Only the icons a build uses ship: the build finds static names in templates
(`name="trash"`, `icon="plus"`) and `icon: '…'` properties in scripts, and
fails with a sentence naming the file for a name Lucide does not have. Names
made at run time go in the `icons` option of `weftIcons()`.

Without `label` the icon is decorative (`aria-hidden="true"`); with one it is
an image with that accessible name (`role="img"`). `size` is `sm` (16 px),
`md` (20 px) or `lg` (24 px); without it the icon is 1.25em.
