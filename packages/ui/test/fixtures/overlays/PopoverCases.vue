<script setup vapor lang="ts">
import { ref } from 'vue'
import Popover from '../../../src/overlays/Popover.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = withDefaults(defineProps<{ input?: Input; dir?: 'ltr' | 'rtl'; modal?: boolean; dismissible?: boolean; autoFocus?: boolean; mode?: 'click' | 'hover'; arrow?: boolean; defaultOpen?: boolean }>(), {
  dismissible: true,
  autoFocus: true,
})
const input = ref<Input | undefined>(props.input)
const open = ref(props.defaultOpen ?? false)
const offset = ref(8)
const arrowOn = ref(props.arrow ?? false)
defineExpose({ input, open, offset, arrow: arrowOn })
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
        :mode="mode"
        :arrow="arrowOn"
        :offset="offset"
        :open-delay="0"
        :close-delay="0"
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
