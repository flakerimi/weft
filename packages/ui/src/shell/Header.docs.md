---
group: shell
purpose: The bar at the top of a page: the site's name, its navigation and actions.
use: At the top of a site's pages (a landing page, docs); an app built on AppShell has its header there.
avoid: Inside a card or a dialog, use their own header slots; for an app frame with a sidebar, use AppShell.
adapts:
  input: The narrow menu button is at least --weft-hit-min square on touch (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: When the viewport is narrow and a menu slot is given, the middle gives way to a menu button that opens the menu in a Sheet from the start side.
---
The `title` links to `to` (`/`); the `leading` slot replaces it with a logo.
The default slot is the middle (a NavigationMenu), `actions` the end. The
`menu` slot is what a narrow screen shows in its sheet, with `close()` to
shut it after a link is chosen; bind the sheet with `v-model:open`. Without
a `menu` slot the middle stays when narrow, and a NavigationMenu there takes
its own narrow form. `sticky` (on) keeps it at the top while the page
scrolls.

It is a `<header>`: the page's banner landmark when it is not inside an
article, aside, main, nav or section.
