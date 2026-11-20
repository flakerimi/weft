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
build refuses anything else with a sentence naming the prop. `when="media"`
needs a `media` query. Ids from `useId` inside an island are counted per
island, so they match between the server and the browser.
