---
group: basics
purpose: A button that runs an action, or a link drawn as a button when it has `to`.
use: For the actions on a page, a form's submit, a dialog's confirm; solid for the one main action, soft, outline or ghost for the others.
avoid: For moving to another page within running text, use Link; for a choice that stays on or off, use a switch or checkbox.
adapts:
  input: On touch its hit area grows to at least 44 px (--weft-hit-min) through a pseudo-element, so its drawn size and the layout stay the same; hover colours apply only where the device can hover.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: Nothing changes; set block to fill a narrow column.
---
`variant` is `solid`, `soft`, `outline` or `ghost`; `tone` is `neutral`,
`primary`, `danger`, `warning` or `success`; `size` is `sm`, `md` or `lg`.
Give the text as `label` or as the default slot. With only an `icon` the
button is square: give it an `aria-label`.

**Loading** shows a spinner over the content, which stays in place (hidden),
so the button keeps its width. It sets `aria-busy="true"` and
`aria-disabled="true"`, and blocks activation by mouse, keyboard and form
submit, but stays focusable, so a keyboard user does not lose their place.

**Disabled** is the native `disabled` attribute on a button. With `to` the
button is an `<a>`; disabled, it has no address and is announced as a
disabled link.

**Keyboard:** a native button, so Enter and Space press it; as a link, Enter
follows it. Focus shows the theme's focus ring.

`click` is emitted only when the button is pressed while neither loading nor
disabled. `type` defaults to `button`, so a Button inside a form submits it
only with `type="submit"`.
