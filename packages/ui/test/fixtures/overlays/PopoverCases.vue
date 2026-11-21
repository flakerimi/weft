<script setup vapor lang="ts">
import { ref } from 'vue'
import Popover from '../../../src/overlays/Popover.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = withDefaults(defineProps<{ input?: Input; dir?: 'ltr' | 'rtl'; modal?: boolean; dismissible?: boolean; autoFocus?: boolean }>(), {
  dismissible: true,
  autoFocus: true,
})
const input = ref<Input | undefined>(props.input)
const open = ref(false)
defineExpose({ input, open })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <button class="before">before</button>
      <Popover
        v-model:open="open"
        class="mine"
        title="Filters"
        :modal="modal"
        :dismissible="dismissible"
        :auto-focus="autoFocus"
      >
        <button class="trigger">Filters</button>
        <template #content="{ close }">
          <input class="field" />
          <button class="done" @click="close()">Done</button>
        </template>
      </Popover>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
