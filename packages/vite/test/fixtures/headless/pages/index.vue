<script setup vapor lang="ts">
// Every headless composable, used as a component uses it, rendered on the server.
import { computed, ref, useTemplateRef } from 'vue'
import {
  overlayStack,
  useControllable,
  useDismiss,
  useFloating,
  useFocusTrap,
  useId,
  useRovingFocus,
  useScrollLock,
  useTypeahead,
} from '@weft/ui'

const props = defineProps<{ modelValue?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const value = useControllable(props, 'modelValue', emit, 'first')
const open = ref(true)
const id = useId('menu')
const trigger = useTemplateRef<HTMLElement>('trigger')
const panel = useTemplateRef<HTMLElement>('panel')
const { isTop } = useDismiss(panel, { onDismiss: () => (open.value = false), escape: () => true, returnTo: trigger })
useFocusTrap(panel, { active: open, returnTo: trigger })
useScrollLock(open)
const { styles, isPositioned } = useFloating(trigger, panel, { placement: 'bottom-start', offset: 4, flip: true, shift: true, size: true })
const roving = useRovingFocus(panel, { orientation: 'vertical' })
const labels = ['first', 'second']
const typeahead = useTypeahead(() => labels, (i) => roving.focus(i))
const onTop = computed(() => overlayStack.isTop('none'))
</script>

<template>
  <div>
    <button ref="trigger" type="button" :aria-controls="id" :aria-expanded="open">{{ value }}</button>
    <ul v-if="open" :id="id" ref="panel" role="menu" :style="styles" :data-positioned="isPositioned" :data-top="isTop || onTop" @keydown="typeahead.onKeydown($event)">
      <li v-for="l in labels" :key="l" role="menuitem" data-weft-item>{{ l }}</li>
    </ul>
  </div>
</template>
