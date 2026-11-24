<script setup vapor lang="ts">
// The site's copy button, an island, for one of two things:
// - `command`: a one-line command, shown in a bar with the button beside it.
//   The command is in the server HTML, so it can be read and selected
//   before the island starts.
// - `from`: a block of code or text elsewhere on the page, named by its id;
//   the button alone, which copies that element's text, so the text is sent
//   once, in the page.
// `name`, where "Copy" alone would not say what it copies, is the button's
// name until it is pressed ("Copy prompt"); it still reads Copy, and then
// Copied, which a screen reader is told.
// Until the island starts, and without JavaScript, the button is hidden: it
// could do nothing. Without a clipboard (an insecure origin, a denied
// permission) it selects the text instead.
import { Button } from '@weft/ui'
import { ref } from 'vue'

const props = defineProps<{
  /** The command to show and copy. */
  command?: string
  /** The id of the element whose text it copies, in place of `command`. */
  from?: string
  /** The button's name until it is pressed, saying what it copies: "Copy prompt". */
  name?: string
}>()

const label = ref('Copy')
let reset: ReturnType<typeof setTimeout> | undefined

async function copy(event: MouseEvent) {
  // The command's text is the element before the button.
  const source = props.from ? document.getElementById(props.from) : (event.currentTarget as HTMLElement).previousElementSibling
  try {
    await navigator.clipboard.writeText(props.command ?? source?.textContent ?? '')
    label.value = 'Copied'
  } catch {
    if (source) getSelection()?.selectAllChildren(source)
    label.value = 'Selected'
  }
  clearTimeout(reset)
  reset = setTimeout(() => (label.value = 'Copy'), 2000)
}
</script>

<template>
  <div :class="command !== undefined ? 'copy-command' : 'copy-code'">
    <code v-if="command !== undefined" class="copy-command__text">{{ command }}</code>
    <Button
      class="copy-button"
      :size="command !== undefined ? 'md' : 'sm'"
      :label="label"
      :aria-label="name && label === 'Copy' ? name : undefined"
      :aria-controls="from"
      aria-live="polite"
      @click="copy"
    />
  </div>
</template>

<style>
.copy-command {
  display: flex;
  align-items: center;
  gap: var(--weft-space-3);
  padding: var(--weft-space-2) var(--weft-space-2) var(--weft-space-2) var(--weft-space-4);
  background: var(--weft-color-surface);
  border: var(--weft-border-width, 1px) solid var(--weft-color-border);
  border-radius: var(--weft-radius-control);
}
.copy-command__text {
  flex: 1;
  min-width: 0;
  /* On a narrow screen the command wraps rather than hiding its end. */
  overflow-wrap: anywhere;
  font: 0.85em/1.6 var(--weft-font-mono);
}
.copy-button { flex: none; min-inline-size: 5.5em; }
.copy-command .copy-button { font-size: 0.85em; }
.copy-code { display: contents; }
/* The island runtime takes the props script out as the island starts: until then the button is not offered. */
weft-island:has(> script[type="application/json"]) .copy-button { visibility: hidden; }
</style>
