---
group: feedback
purpose: What a list, a search or a page shows when it has nothing yet, with the actions that would fill it.
use: In a Table's empty slot, for a new account with no records, for a search with no results; it replaces the hand-built empty states of admin panels such as Negenet's.
avoid: When loading failed, use ErrorState; while loading, use Skeleton rows or the Table's loading state.
adapts:
  input: On touch its buttons are at least --weft-hit-min tall (Button's rule).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: The actions wrap and stay centred when its box is narrow; the text keeps a readable measure.
---
`title` says what is empty, `description` what to do about it, `icon` (or
the leading slot) sits in a disc above them. The default slot adds more
content, the `footer` slot a help link. `actions` are Buttons given as data
(`{ label, onClick, to, icon, tone, variant }`): the first solid primary,
the rest outlined; the actions slot replaces them.

The title is a heading at `level` (2 to 6) when given, so it takes its
place in the page's outline; without a level it is bold text. `variant` is
`outline` (dashed border, default), `soft` (tinted) or `ghost`; `size`
sets its space and icon. `loading` turns the icon into a spinner and makes
it `aria-busy`, for a search still running.
