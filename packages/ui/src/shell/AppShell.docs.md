---
group: shell
purpose: The frame of an app: navigation, a header and the main content, laid out for the screen.
use: As the root of an app's pages (a dashboard, an admin, a client portal).
avoid: For a marketing or docs site, use Header and Footer around the page; for a panel that slides in, Sheet.
adapts:
  input: On a narrow touch screen the navigation is a bottom bar of up to five links, icon over label, at least 56 px tall, the rest under More in a bottom Sheet; on a narrow pointer screen it is in a Sheet opened from the header's menu button.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Regular and wide viewports get a sidebar beside the header and content, collapsible to icons; narrow ones (under 640 px) get the bottom bar (touch) or the menu sheet (pointer). A change closes an open sheet and keeps focus on the same item.
---
`items` are `NavItem`s (see NavigationMenu), or groups of them; give each an
`icon`, which is all a collapsed sidebar and the bottom bar show besides the
label. `title` names the app at the top of the sidebar (and in the header
when there is no sidebar), linking to `to`; the `brand` slot replaces it.
Slots: `header` (the header's middle), `actions` (its end), `sidebar` (in
place of the navigation built from `items`), `footer` (the bottom of the
sidebar, and of the menu sheet), and the default slot, the page in `<main>`.
It emits `select(item, event)` for an item chosen anywhere; sheets close
when an item is chosen.

Inside a WeftProvider marked `inset` (a preview drawn inside a page) the
default slot goes in a `<div>`, so that page keeps its one `<main>`.

**Collapsing.** The header's toggle, named "Sidebar" (`labels.sidebar`),
collapses the sidebar to icons and back, labels in tooltips: a button the
keyboard reaches, whose `aria-expanded` says the state, with
`aria-controls`. Bind it with `v-model:collapsed`. The state is kept in
localStorage under `persist` (`weft:app-shell:collapsed`; false keeps
nothing), read after mounting and never on the server. So that a reload
does not flash the sidebar open, AppShell registers a pre-paint snippet
(`usePrePaint`): on the pages that render an AppShell the page build writes
a small blocking inline script at the top of `<head>`, which copies the
stored state to `<html data-weft-sidebar="collapsed">`, and CSS draws the
rail from the first paint. The server's markup is the same either way, so
hydration matches.

**Layouts.** Once mounted it reads Weft's input and the viewport's width and
sets `data-layout`: `sidebar`, `bar` or `drawer`. The server renders every
part, and until mounting CSS picks the layout from the viewport
(`max-width: 639.98px`, with `pointer: coarse` for the bar), so a page
that is not hydrated lays out right too; its bottom bar links work without
JavaScript, the sheets and the toggle need it. Once mounted only the
layout's own parts render, so the `brand`, `sidebar` and `footer` slots are
drawn once; before mounting the `brand` slot is drawn twice (in the sidebar
and in the header), so give it no ids. When the layout changes, an open
drawer or More sheet closes, and focus in a part that goes moves to the same
item in the new layout, else to its menu button, toggle or bar. Current
pages say `aria-current="page"` (Link's rule).

**Bottom bar.** Links without children fill it, in order; when there are
more than five, or any item is a group or a button, it shows the first four
links and a More button, whose sheet lists the rest.
