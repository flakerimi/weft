<script setup vapor lang="ts">
import { ref } from 'vue'

const props = defineProps<{ command: string }>()

// The command is in the server HTML, so it can be read and selected before this
// island starts; the button copies it once it has.
const label = ref('Copy')
let reset: ReturnType<typeof setTimeout> | undefined

async function copy(e: MouseEvent) {
  try {
    await navigator.clipboard.writeText(props.command)
    label.value = 'Copied'
  } catch {
    // No clipboard (an insecure origin, a denied permission): select it instead.
    const text = (e.currentTarget as HTMLElement).previousElementSibling
    if (text) getSelection()?.selectAllChildren(text)
    label.value = 'Selected'
  }
  clearTimeout(reset)
  reset = setTimeout(() => (label.value = 'Copy'), 2000)
}
</script>

<template>
  <div class="copy-command">
    <code class="copy-command__text">{{ command }}</code>
    <button type="button" class="copy-command__button" aria-live="polite" @click="copy">{{ label }}</button>
  </div>
</template>

<style>
.copy-command {
  display: flex;
  align-items: center;
  gap: var(--weft-space-3);
  padding: var(--weft-space-2) var(--weft-space-2) var(--weft-space-2) var(--weft-space-4);
  background: var(--weft-color-surface);
  border: 1px solid var(--weft-color-border);
  border-radius: var(--weft-radius-control);
}
.copy-command__text {
  flex: 1;
  min-width: 0;
  /* On a narrow screen the command wraps rather than hiding its end. */
  overflow-wrap: anywhere;
  font: 0.85em/1.6 var(--weft-font-mono);
}
.copy-command__button {
  flex: none;
  min-height: var(--weft-control-height);
  min-width: 5.5em;
  padding: 0 var(--weft-space-4);
  font: inherit;
  font-size: 0.85em;
  font-weight: var(--weft-weight-title);
  color: var(--weft-color-primary-fg);
  background: var(--weft-color-primary);
  border: 0;
  border-radius: calc(var(--weft-radius-control) - 4px);
  cursor: pointer;
  transition: filter var(--weft-duration-control) var(--weft-ease-standard);
}
.copy-command__button:hover { filter: brightness(1.1); }
.copy-command__button:active { filter: brightness(0.95); }
</style>
