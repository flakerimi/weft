<script setup vapor lang="ts">
import { ref } from 'vue'
import Tooltip from '../../../src/overlays/Tooltip.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ input?: Input }>()
const input = ref<Input | undefined>(props.input)
const open = ref(false)
defineExpose({ input, open })
</script>

<template>
  <WeftProvider :input="input">
    <Tooltip v-model:open="open" text="Delete file" :delay="100">
      <button class="icon-only" aria-describedby="hint"><svg aria-hidden="true"></svg></button>
    </Tooltip>
    <Tooltip text="Saves a draft" :delay="0">
      <button class="labelled">Save</button>
    </Tooltip>
    <Tooltip text="Copy link" :delay="0">
      <button class="named" aria-label="Copy"><svg aria-hidden="true"></svg></button>
    </Tooltip>
    <span id="hint">Cannot be undone.</span>
    <button class="elsewhere">elsewhere</button>
  </WeftProvider>
</template>
