<script setup vapor lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import type { Gap } from './types.ts'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

const props = withDefaults(defineProps<{ cols?: Cols; gap?: Gap }>(), {
  cols: () => ({ narrow: 1, regular: 2, wide: 3 }),
  gap: 4,
})

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const count = computed(() => {
  if (typeof props.cols === 'number') return props.cols
  const c = props.cols
  const fallback = { narrow: 1, regular: c.narrow ?? 2, wide: c.regular ?? 3 }
  return c[width.value] ?? fallback[width.value]
})
</script>

<template>
  <div ref="root" class="weft-grid" :data-gap="gap" :data-width="width" :style="{ '--weft-grid-cols': count }">
    <slot />
  </div>
</template>
