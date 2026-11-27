---
group: basics
purpose: A button that runs an action, or a link drawn as a button when it has `to`.
use: For the actions on a page, a form's submit, a dialog's confirm; solid for the one main action, soft, outline or ghost for the others.
avoid: For moving to another page within running text, use Link; for a choice that stays on or off, use a switch or checkbox.
adapts:
  input: On touch an md button is at least 44 px tall (--weft-hit-min); a small or square button keeps its drawn size and grows its hit area to 44 px through a pseudo-element. Hover colours apply only where the device can hover.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; set block to fill a narrow column.
---
`variant` is `solid`, `soft`, `outline` or `ghost`; `tone` is `neutral`,
`primary`, `danger`, `warning` or `success`; `size` is `sm`, `md` or `lg`.
Give the text as `label` or as the default slot; the `leading` and
`trailing` slots replace `icon` and `trailing-icon`. With only an icon the
button is square (`square` makes any button square): give it an
`aria-label`, or put it in a `Tooltip`, whose text names it; otherwise it
warns once in development. `active` draws it as the
current one of a set; with `to` it also sets `aria-current="page"`.

**Touch:** an md button grows to 44 px tall on touch, so the visible target
is the hit target. A small or square button keeps its size (a toolbar would
change shape otherwise) and its `::before` reaches 44 px instead.

**Loading** shows a spinner (or `loading-icon`, turning) over the content,
which stays in place (hidden), so the button keeps its width. It sets `aria-busy="true"` and
`aria-disabled="true"`, and blocks activation by mouse, keyboard and form
submit, but stays focusable, so a keyboard user does not lose their place.

**Disabled** is the native `disabled` attribute on a button. With `to` the
button is an `<a>`, taking `rel` and `target`, with `noopener` added for
another site; disabled, it has no address and is announced as a disabled
link; loading, it keeps its address (and focus) but neither a click, a
middle click nor Enter follows it.

**Keyboard:** a native button, so Enter and Space press it; as a link, Enter
follows it. Focus shows the theme's focus ring.

**In a Form.** As a link it is disabled while the Form is disabled or busy,
so a Cancel cannot navigate mid-submit (a disabled fieldset disables native
buttons but leaves links alone). While the Form is busy, a `type="submit"`
button loads, keeping focus, and any other button works as ever, so a
Cancel or Stop can abort the submit (as in Nuxt UI).

**By permission.** With `perm`, such as `perm="products:edit"`, the button
is drawn only for a signed-in user who has that permission
(`useSession().can`, always true for the role `owner`). It hides; it does not
disable, and it guards nothing: the route behind it must still check, as
`auth.Per` does. Pages are built with no session, so the server never draws a
`perm` button, and neither does the browser's first render: it appears once
the button has mounted and `GET /api/auth/me` has answered, asked once for
the whole page, so hydration always matches. Without `perm` nothing is asked.

`click` is emitted only when the button is pressed while neither loading nor
disabled. `type` defaults to `button`, so a Button inside a form submits it
only with `type="submit"`.
