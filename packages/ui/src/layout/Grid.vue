<script setup vapor lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import type { Gap } from './types.ts'
import '../styles/base.css'
import '../styles/gap.css'
import '../styles/grid.css'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

const props = withDefaults(defineProps<{
  /** Columns: one number for every width, or one per width class of the grid's own width. */
  cols?: Cols
  /** Space between cells, from the spacing scale (`--weft-space-*`). */
  gap?: Gap
}>(), {
  cols: () => ({ narrow: 1, regular: 2, wide: 3 }),
  gap: 4,
})

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const widthId = width.id
const order = ['narrow', 'regular', 'wide'] as const
const defaults = { narrow: 1, regular: 2, wide: 3 }

function colsAt(w: (typeof order)[number]): number {
  if (typeof props.cols === 'number') return props.cols
  const c = props.cols
  for (let i = order.indexOf(w); i >= 0; i--) {
    const n = c[order[i]!]
    if (n !== undefined) return n
  }
  return defaults[w]
}

const narrowCols = computed(() => colsAt('narrow'))
const regularCols = computed(() => colsAt('regular'))
const wideCols = computed(() => colsAt('wide'))
// A partial object falls back to the nearest smaller width given, else the defaults.
const count = computed(() => colsAt(width.value))
</script>

<template>
  <div
    ref="root"
    class="weft-grid"
    :data-gap="gap"
    :data-width="width"
    :data-weft-w="widthId"
    :style="{
      '--weft-grid-cols': count,
      '--weft-grid-cols-narrow': narrowCols,
      '--weft-grid-cols-regular': regularCols,
      '--weft-grid-cols-wide': wideCols,
    }"
  >
    <slot />
  </div>
</template>
