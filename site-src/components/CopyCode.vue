<script setup vapor lang="ts">
// The site's copy button for a block of code, an island: it copies the text
// of the element `from` names (a code block on the page), so the code is
// sent once, in the page. Until the island starts, and without JavaScript,
// it is hidden: it could do nothing. CopyCommand is the one-line command's.
import { ref } from 'vue'

const props = defineProps<{
  /** The id of the element whose text it copies. */
  from: string
}>()

const label = ref('Copy')
let reset: ReturnType<typeof setTimeout> | undefined

async function copy() {
  const source = document.getElementById(props.from)
  try {
    await navigator.clipboard.writeText(source?.textContent ?? '')
    label.value = 'Copied'
  } catch {
    // No clipboard (an insecure origin, a denied permission): select it instead.
    if (source) getSelection()?.selectAllChildren(source)
    label.value = 'Selected'
  }
  clearTimeout(reset)
  reset = setTimeout(() => (label.value = 'Copy'), 2000)
}
</script>

<template>
  <button type="button" class="copy-code" :aria-controls="from" aria-live="polite" @click="copy">{{ label }}</button>
</template>

<style>
.copy-code {
  min-block-size: max(var(--weft-control-sm, 28px), var(--weft-hit-min, 24px));
  min-inline-size: 5em;
  padding: 0 var(--weft-space-3);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: var(--weft-weight-title);
  color: var(--weft-color-primary-fg);
  background: var(--weft-color-primary);
  border: 0;
  border-radius: var(--weft-radius-control);
  cursor: pointer;
}
.copy-code:hover { background: var(--weft-color-primary-hover); }
.copy-code:active { background: var(--weft-color-primary-active); }
/* The island runtime takes the props script out as the island starts: until then the button is not offered. */
weft-island:has(> script[type="application/json"]) > .copy-code { visibility: hidden; }
</style>
