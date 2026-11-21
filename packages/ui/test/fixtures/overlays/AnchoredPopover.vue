<script setup vapor lang="ts">
// A Popover with no trigger, floating against an element the app owns.
import { ref, useTemplateRef } from 'vue'
import Popover from '../../../src/overlays/Popover.vue'

const open = ref(true)
const field = useTemplateRef<HTMLElement>('field')
defineExpose({ open })
</script>

<template>
  <input ref="field" class="field" />
  <Popover v-model:open="open" :anchor="field ?? undefined" :auto-focus="false" title="Suggestions">
    <template #content><p class="inside">Suggestions</p></template>
  </Popover>
  <button class="after">after</button>
</template>
