---
group: shell
purpose: The end of a page: copyright, secondary links, social links.
use: At the bottom of a site's pages.
avoid: Inside a card or a dialog, use their own footer slots.
adapts:
  input: Nothing changes between pointer and touch; links inside bring their own touch sizes (Link with `standalone`).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: When its own width is narrow (under 640 px) the row stacks its parts, centred. Before it mounts, and on a page that is never hydrated, a viewport under 640 px does.
---
Slots: `top` (a band above, such as columns of links), `leading` (the start
of the row), the default slot (its middle), `trailing` (its end) and
`bottom` (a band below). It is a `<footer>`: the page's contentinfo landmark
when it is not inside an article, aside, main, nav or section. It has
nothing interactive of its own.
