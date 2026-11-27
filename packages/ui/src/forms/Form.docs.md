---
group: forms
purpose: A form that checks its values against a Weft field schema before it submits.
use: For any form whose values need checking before they are sent, with the same rules the server enforces.
avoid: For a search box or a single setting that takes effect at once, a Form is not needed.
adapts:
  input: Nothing of its own; the fields inside it adapt.
  surface: Nothing changes in behaviour.
  width: Nothing changes; its fields stack in a column.
---
`state` is the values, an object the fields bind into with `v-model`;
`schema` is a Weft field schema - plain JSON a server can send, the same
shape `weft gen` writes as `schema` from the Go types - keyed by field name:

    { email: { required: true, email: true }, age: { min: 18, message: 'You must be 18 or over.' } }

Each rule is optional: `required`, `min` and `max` (a length for text and
lists, bounds for numbers), `pattern` (matched against the whole text, as the
HTML attribute does), `email`, `url` (an absolute `http:` or `https:`
address with a host: "Enter a web address that starts with http:// or
https://."), `enum` (a list of the values allowed: "Choose one of: a, b."),
and `message` in place of the default sentences. `required` counts text of
only spaces as empty; every other rule skips only a missing or empty value
(`undefined`, `null`, `''` or an empty list), so spaces alone fail `enum`,
`url` and `email`. Its TypeScript type is `FieldSchema`, and
`validateFields` checks values against one anywhere.

**When it checks.** On submit every field is checked. With errors, `error`
fires with them, each FormField with a matching `name` shows its error, and
focus moves to the first invalid control; without, the `@submit` handler
runs with a deep copy of the values (lists and objects too), typed as
`state` is. A field the schema marks `nullable` (a pointer in Go, which
`weft gen`'s schema marks) is sent as `null` when emptied, `''` included,
which clears it; without the schema there is no way to tell, so every other
`null` (an emptied number, date or choice) is left out of that copy, and
the server reads the field as not sent. A `required` field is stopped by
the rule first.
After the first submit, a field is checked again when it is
left, and a field showing an error is checked as it changes, so the error
goes the moment it is fixed - but no new error appears while typing.

**Server errors.** A template ref's `setErrors({ email: 'That address is
taken.' })` shows errors from elsewhere. Leaving such a field keeps its error
(the server said so, and nothing changed); changing the field checks it
again. `validate()` checks without submitting, and `clear()` clears every
error.

A server error on a field the schema has no rule for clears the first time
that field changes: there is nothing to check it against.

**Its template ref.** `FormApi` is the ref's type:
`useTemplateRef<FormApi>('form')` gives `setErrors(errors)`, `validate()`,
`clear()`, `submit()` (as the submit button would), and `errors` and
`loading` read as values.

**Disabled.** The content sits in a `<fieldset>` with no frame, so
`disabled` disables every control in the form at once - Weft's and native
ones, Buttons included - and a submit does nothing. A fieldset leaves links
alone, so a Link, or a Button with `to`, in the form reads it and is
disabled as a disabled link is: no address, `aria-disabled`, no navigation.

**Busy.** `loading`, or with `loading-auto` (on by default) a `@submit`
handler that returns a promise, makes the form busy until it settles: it sets
`aria-busy`, a second press or Enter submits nothing, its fields are
disabled and its links have no address. The submit button is not natively
disabled, so it keeps focus: it shows its spinner, is `aria-busy` and
`aria-disabled`, and ignores presses. Any other button works, so a Cancel or
Stop can abort the submit (as in Nuxt UI). Raw native controls in the form (a
plain `<input>` or `<button>`) are not blocked while busy, only while
`disabled`; use Weft's components to have them follow the form. A field that
had focus (Enter pressed in it, or focused as `loading` turned on) gets it
back when the form is done. (`@submit` is a handler prop, as in Nuxt UI, so
the form can wait for it.) The native form has `novalidate`: Weft shows the
errors, not the browser.

**Before it is live.** A Form sends only through its `@submit` handler,
so it belongs in an island: one drawn by the page, with no island around
it, is never live and does nothing on submit; in development its first
submit says so in the console. Until its island is live it
sends nothing: a small inline script at the top of the page's `<head>`
stops every submit of a Weft form from the first moment, and the form is
`method="post"`, so with scripts off its own submit puts nothing in the
address (a submit button the app gives `formmethod="get"` still does).
Its fields stay usable meanwhile: what a person types, or a password
manager fills, before the island is live is kept when it hydrates and
reaches the model. An app's own `form.submit()` fires no submit event, by
the platform's design, so it bypasses the script and sends a POST to the
page.

A site with a Content-Security-Policy must allow the page's inline
pre-paint script, `<script data-weft-prepaint>`, and its inline
`<script type="speculationrules">`, by their SHA-256 hashes or with
`'unsafe-inline'` (a nonce needs a server that writes one into each answer,
which a built page does not have). The build wraps each pre-paint snippet in
`(function(){…})()`, so a page whose only pre-paint component is Form has
exactly (`'sha256-tx8/QjztXL6iQA+dqwGLMSVBWxKaWIfL8huD/H0v4SY='`)

    (function(){window.addEventListener("submit",function(e){var t=e.target;if(t&&t.classList&&t.classList.contains("weft-form"))e.preventDefault()},true)})()

and a page whose other components register pre-paint code too (Grid or the
landing blocks Hero, FeatureGrid, Pricing, Testimonials, Faq and Cta, a
horizontal Stepper or Timeline or a vertical Tabs, via `useWidth`, AppShell
unless `persist` is `false`, or a closable Banner with an `id`) has each
wrapped snippet joined into the same script with `;\n`, one per line (the
width snippet appears at most once per page, whatever the number of width
components). Because the script's text — and so its hash — changes with the
Weft version, which components a page renders, the order they register in, and
each AppShell `persist` key or Banner `id`, a strict Content-Security-Policy
needs one `'sha256-…'` entry per distinct inline script across the site's
built pages, collected again after each build or `weft upgrade`. Run this in
the folder that holds `dist/` (`web/` in an app) to print every hash, quoted
for `script-src`, one per line:

    node -e 'const fs=require("node:fs"),crypto=require("node:crypto");console.log([...new Set(fs.globSync("dist/**/*.html").flatMap(f=>[...fs.readFileSync(f,"utf8").matchAll(/<script (?:data-weft-prepaint|type="speculationrules")>([\s\S]*?)<\/script>/g)].map(m=>"\x27sha256-"+crypto.createHash("sha256").update(m[1]).digest("base64")+"\x27")))].join("\n"))'
