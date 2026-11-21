---
group: basics
purpose: A key on the keyboard, for showing shortcuts.
use: In help text, menus and tooltips, one Kbd per key of a shortcut.
avoid: For code or a value to copy, use a code element.
adapts:
  input: Nothing changes; show shortcuts only where a keyboard is likely, which the page decides.
  surface: Nothing changes; text follows the surface's type scale.
  width: Nothing changes.
---
`value` is a character (`K`) or a key name. Names with a symbol draw the
symbol and say the name to assistive technology, which reads symbols poorly:
`command` (⌘), `shift` (⇧), `option` (⌥), `enter` (↵), `backspace` (⌫),
`tab` (⇥) and the arrows `arrowup`, `arrowdown`, `arrowleft`, `arrowright`.
Any other value is drawn as given. It draws what it is told and never guesses
the reader's operating system, so the server and browser agree. The default
slot replaces the value.
