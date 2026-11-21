# Weft's headless layer

Behaviour without markup or styles, shared by every component. Each file's
header comment is its contract; this page only says where to look. Nothing
here touches the DOM at import or during setup, so all of it runs on the
server.

| File | What it does | Read its header for |
| --- | --- | --- |
| `overlayStack.ts` | The one ordered list of open overlays; Escape and outside presses go to the top one. | `push`/`pop`, `passThrough` entries (a tooltip passes a press on to the overlay below), `returnFocus: false` (an overlay that never takes focus returns none), `returnTo`, and why the stack alone returns focus. |
| `useDismiss.ts` | Close on Escape or a press outside; joins and leaves the stack with its element. | `ignore` (the trigger), `returnTo`, `returnFocus`, `passThrough`, the microtask that keeps a popover's place when it becomes a sheet. |
| `useFocusTrap.ts` | Keep focus inside a modal overlay. | `initial`, `firstFocus` (below), `data-weft-trap-exempt`, nested traps, `returnFocus` as a getter, moving the trap to a re-rendered element. |
| `dom.ts` | DOM helpers for the others. | `firstFocus(root)`: the tabbable marked `data-weft-autofocus`, else the first; `tabbables`, `restoreFocus`, `isRtl`. |
| `useInert.ts` | Make the page behind a modal inert. | Holders counted across the page (the last to let go restores it), `#weft-overlays` left reachable, hold after the opening render, release at once. |
| `useScrollLock.ts` | Stop the page behind an overlay scrolling. | Locks counted across the page, the scrollbar padding, inline styles put back exactly. |
| `useRovingFocus.ts` | One tab stop for a group, moved with the arrows (APG roving tabindex). | `orientation` (`grid` and `columns`), `loop`, `rtl`, `tabStops: 'all'` (arrows, but every item keeps its tab stop), `inner: 'ignore'` (keys in a control inside an item stay with that control). |
| `useTypeahead.ts` | Type to jump to an item by its label. | Labels as a getter, `''` for items that never match, `reset()`. |
| `useVirtualList.ts` | Render only the rows in view of a long list. | Fixed row height, `pinned`, `scrollTo`, the server's `viewport`, `leading` (a caption and header before row 0) and `stuck` (a sticky header over the view's top). |
| `useFloating.ts` | Position a floating element against its reference (`@floating-ui/dom`). | `placement`, `offset`, `flip`, `shift`, `size`, `strategy`, when it starts and stops. |
| `useControllable.ts` | The value behind a `v-model`, bound or not. | Bound and unbound writes, the read-back until the parent answers, boolean models defaulting to `undefined`. |
| `useId.ts` | Ids that match between server and browser. | The per-island `ID_SCOPE`, one call per id in setup. |
| `swipe.ts` | Follow a drag towards an edge (a drawer, a toast). | `direction()` read at drag start, `move` and `end` (`{ distance, speed }`). |

`index.ts` exports the public ones.
