---
group: basics
purpose: A link to another page or site, marking itself when it is the current page.
use: For navigation, in running text, in a list of pages or a footer.
avoid: For an action that changes something, use Button; for a link that should look like a button, use Button with to.
adapts:
  input: A standalone link (in a list, a footer, a toolbar) gets a hit area of at least 44 px on touch through a pseudo-element; a link in running text keeps its text size and no larger hit area, which would cover the lines around it. Hover styles apply only where the device can hover.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes.
---
**Standalone or in text.** Set `standalone` on a link that stands on its
own: in navigation, a list, a footer, a toolbar. It becomes an inline box
whose hit area reaches `--weft-hit-min` (44 px on touch). Leave it unset in
running text: a larger hit area there would cover the line above and below
and take taps meant for the links on them (WCAG 2.5.8 exempts targets in a
sentence).

The text is the theme's `--weft-color-primary-text`, which it checks for
4.5:1 on the page.

An external link (an address with a scheme and host, `https://…` or `//…`,
or any with `external`) gets `rel="noopener"`, added to any `rel` you give.
`target` passes through as the native attribute.

The link to the page the browser is on gets `aria-current="page"`. It is
worked out in the browser after mounting, so the server's markup and the
first client render agree; on a page with no JavaScript the server cannot
know, so set `active` where the layout knows the current page. `active`
always wins over the comparison.

**Disabled:** no address and no target, so it leaves the tab order;
announced as a disabled link (`role="link"`, `aria-disabled="true"`); a click
does nothing.

**Keyboard:** a native anchor, so Tab reaches it and Enter follows it.
