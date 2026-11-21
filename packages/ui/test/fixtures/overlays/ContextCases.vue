<script setup vapor lang="ts">
import { ref } from 'vue'
import ContextMenu from '../../../src/overlays/ContextMenu.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { MenuItem } from '../../../src/overlays/menu.ts'

const props = defineProps<{ input?: Input; disabled?: boolean }>()
const input = ref<Input | undefined>(props.input)
const log = ref<string[]>([])
const items: MenuItem[] = [
  { label: 'Copy', onSelect: () => log.value.push('copy') },
  { label: 'Paste', onSelect: () => log.value.push('paste') },
]
defineExpose({ input, log })
</script>

<template>
  <WeftProvider :input="input">
    <ContextMenu :items="items" :disabled="disabled" aria-label="Clipboard">
      <div class="area" tabindex="0">Right-click here</div>
    </ContextMenu>
    <button class="after">after</button>
  </WeftProvider>
</template>
