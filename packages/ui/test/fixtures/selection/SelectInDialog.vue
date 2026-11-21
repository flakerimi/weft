<script setup vapor lang="ts">
import { ref } from 'vue'
import Dialog from '../../../src/overlays/Dialog.vue'
import Select from '../../../src/selection/Select.vue'

const open = ref(false)
const value = ref<string | undefined>()
defineExpose({ open, value })
</script>

<template>
  <Dialog v-model:open="open" title="Settings">
    <button class="opener">Settings</button>
    <template #body>
      <Select v-model="value" :items="['Small', 'Medium', 'Large']" value-key="" aria-label="Size" />
    </template>
  </Dialog>
</template>
