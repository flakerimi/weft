<script setup vapor lang="ts">
// A key on the keyboard, for shortcuts in text and menus. Named keys with a
// symbol (command, shift, enter, the arrows) draw the symbol and say the name
// to assistive technology, which reads symbols poorly. Platform-neutral: it
// draws what it is given, never guessing the reader's operating system.
import { computed } from 'vue'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/kbd.css'

const props = withDefaults(
  defineProps<{
    /** The key: a character (`K`) or a name with a symbol (`command`, `shift`, `option`, `enter`, `backspace`, `tab`, `arrowup`…). The default slot replaces it. */
    value?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
  }>(),
  { size: 'md' },
)

const KEYS: Record<string, [symbol: string, name: string]> = {
  command: ['⌘', 'Command'],
  cmd: ['⌘', 'Command'],
  shift: ['⇧', 'Shift'],
  option: ['⌥', 'Option'],
  enter: ['↵', 'Enter'],
  backspace: ['⌫', 'Backspace'],
  tab: ['⇥', 'Tab'],
  arrowup: ['↑', 'Up arrow'],
  arrowdown: ['↓', 'Down arrow'],
  arrowleft: ['←', 'Left arrow'],
  arrowright: ['→', 'Right arrow'],
}

const named = computed(() => {
  const key = props.value?.toLowerCase()
  return key && Object.hasOwn(KEYS, key) ? KEYS[key] : undefined
})
</script>

<template>
  <kbd class="weft-kbd" :data-size="size">
    <slot>
      <template v-if="named">
        <span aria-hidden="true">{{ named[0] }}</span>
        <span class="weft-visually-hidden">{{ named[1] }}</span>
      </template>
      <template v-else>{{ value }}</template>
    </slot>
  </kbd>
</template>
