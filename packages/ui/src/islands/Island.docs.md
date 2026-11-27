---
group: islands
purpose: Renders a component on the server and hydrates it in the browser when triggered.
use: For the few interactive parts of an otherwise static page, a counter, a form, a menu, so only they ship JavaScript.
avoid: For content that never changes in the browser, render the component directly; a page with no island ships no script at all.
adapts:
  input: Nothing changes; the island's component adapts as it would anywhere.
  surface: Nothing changes.
  width: Nothing changes.
---
The component must be a `<script setup>` single-file component: its file name
is the island's id. `props` travel to the browser as JSON, so they may hold
only strings, finite numbers, booleans, null, arrays and plain objects; the
build refuses anything else with a sentence naming the prop, `-0` (which
JSON writes as `0`), a symbol key and an array's members other than its
items (which JSON leaves out) included. `when="media"`
needs a `media` query. Ids from `useId` inside an island are counted per
island, so they match between the server and the browser.

Ids are counted by the island's position among the page's islands, the
same on the server (render order) and in the browser (document order). So an
island inside a `<Teleport>`, whose markup lands elsewhere in the document,
is not supported: its position, and so its ids, would differ.

What a person does to the island's form controls before it hydrates is
kept: text typed or filled into a field, a box ticked, a radio or an option
chosen. Just before the mount each such control is set back to what the
server drew, so the hydration matches, and once mounted it gets the
person's value again with an `input` and a `change` event, as the person's
own change would, so `v-model` and a Form's state receive it. A radio
group counts as one value, whichever radio comes first. A control nobody
touched keeps the value its model started with: untouched means as a form
reset would leave it, so a select showing its first option because its
model matches none is untouched.

An island inside another island (a component that uses one, shown in an
island) is not an island of its own: it draws its component in place, and
the outer island hydrates it with the rest of itself.

The page inlines the CSS of everything the island's component uses, not
only what it draws on the server, since the browser may draw the rest later
(a menu that opens, a field that appears). A name taken from the `@weft/ui`
index, `import { Button } from '@weft/ui'`, brings that component's
stylesheet and those of what it draws with, not every component's: the
build follows the names through the index, as a bundler's tree-shaking does
(`@weft/ui` declares its scripts free of side effects in its package.json).
`import * as Weft from '@weft/ui'` takes every name, and so every
stylesheet. Outside islands the page carries only the CSS of the components
it renders.

It has no stylesheet of its own: `<weft-island>` is an unstyled element around the component.
