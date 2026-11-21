---
group: navigation
purpose: Where the page sits in the site, as links from the top down.
use: On pages two or more levels deep (docs, a shop's categories, a file browser), above the page's title.
avoid: For steps of a task, use Stepper; for the site's main links, NavigationMenu.
adapts:
  input: Each link has a hit area of at least --weft-hit-min on touch, without changing how it looks.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The path stays on one line and scrolls sideways when it is too long for its width.
---
`items` are `{ label, to, target, icon, disabled, active }`, from the top;
an item without `to` is text. The `item` slot replaces an item's content and
the `separator` slot the chevron (or give `separator-icon`, a Lucide name).
`label` names the landmark (`Breadcrumb`).

It follows the WAI-ARIA APG Breadcrumb pattern: a `nav` landmark holding an
ordered list. The last item is the current page, `aria-current="page"`,
whether it is a link or text; the others are compared with the browser's
address as Link does, after mounting. Separators are hidden from assistive
technology and mirror right to left.
