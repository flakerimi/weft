---
group: islands
purpose: Renders a component on the server and makes it interactive in the browser, on load, when idle, when visible or when a media query matches.
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
JSON writes as `0`) and a symbol key (which JSON leaves out) included. `when="media"`
needs a `media` query. Ids from `useId` inside an island are counted per
island, so they match between the server and the browser.

Ids are counted by the island's position among the page's islands, the
same on the server (render order) and in the browser (document order). So an
island inside a `<Teleport>`, whose markup lands elsewhere in the document,
is not supported: its position, and so its ids, would differ.

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
