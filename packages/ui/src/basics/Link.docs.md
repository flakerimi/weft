---
group: basics
purpose: A link to another page or site, marking itself when it is the current page.
use: For navigation, in running text, in a list of pages or a footer.
avoid: For an action that changes something, use Button; for a link that should look like a button, use Button with to.
adapts:
  input: On touch its hit area grows to at least 44 px (--weft-hit-min) through a pseudo-element, without changing the text; hover styles apply only where the device can hover.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes.
---
An external link (an address with a scheme and host, `https://…` or `//…`,
or any with `external`) gets `rel="noopener"`, added to any `rel` you give.
`target` passes through as the native attribute.

The link to the page the browser is on gets `aria-current="page"`. It is
worked out in the browser after mounting, so the server's markup and the
first client render agree; on a page with no JavaScript the server cannot
know, so set `active` where the layout knows the current page. `active`
always wins over the comparison.

**Disabled:** no address, so it leaves the tab order; announced as a disabled
link (`role="link"`, `aria-disabled="true"`); a click does nothing.

**Keyboard:** a native anchor, so Tab reaches it and Enter follows it.
