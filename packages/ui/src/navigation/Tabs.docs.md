---
group: navigation
purpose: Views of one thing, one shown at a time, picked from a row of tabs.
use: For settings pages, a record's details and its history, code in several languages.
avoid: To move between pages, use NavigationMenu or links; for sections a reader may open together, use Accordion.
adapts:
  input: Tabs are at least --weft-hit-min tall on touch; a row too long for its width scrolls sideways.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: A vertical tab list goes above the panels when the tabs' own width is narrow (under 640 px), and turns horizontal. CSS does it from `data-width`, which the page's pre-paint script sets, so a page that is never hydrated adapts too; `aria-orientation` and the arrow keys follow once hydrated.
---
`items` are `{ label, value, icon, badge, content, disabled, slot }`; a tab's
value is its `value`, else its index. Bind the selected tab with `v-model`.
The `leading`, default and `trailing` slots replace a tab's icon, label and
badge; the `content` slot a panel's content, and a slot named by an item's
`slot` renders that item's panel. `content: false` draws the tab list alone.
`variant` is `pill` (a tinted track, the default) or `link` (an underline);
`tone` colours the selected tab.

It follows the WAI-ARIA APG Tabs pattern: a `tablist` (named by the
`aria-label` or `aria-labelledby` given to Tabs) with `aria-orientation`,
`tab`s with `aria-selected` and `aria-controls`, and `tabpanel`s named by
their tab, each focusable (`tabindex="0"`).

**Keyboard.** The tab list is one tab stop, on the selected tab. Left and
Right (mirrored right to left), or Up and Down when vertical, move along it,
wrapping and skipping disabled tabs; Home and End go to the ends. With
`activation="automatic"` (the default) the tab focus lands on is selected;
with `manual`, Enter or Space selects it.

**Panels.** Only the selected panel's content is rendered (lazy). With
`keep-alive`, a panel's content stays mounted, hidden, once shown, so a
form's input or a scroll position survives switching away.

**Address.** With `hash`, the address's `#fragment` selects the tab whose
value it names, at mounting and on every `hashchange`, and the selected tab
is written there (replacing the history entry, not adding one). A fragment
naming a disabled tab, or none, is ignored. Off by default. One Tabs per
page owns the hash, the first mounted with `hash`; a second warns once and
leaves the address alone, so give `hash` to one Tabs per page.
