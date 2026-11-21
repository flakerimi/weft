<script setup vapor lang="ts">
import type { Gap } from './types.ts'
import '../styles/base.css'
import '../styles/gap.css'
import '../styles/stack.css'

withDefaults(
  defineProps<{
    /** Lay the children out top to bottom (`column`) or along the line (`row`). */
    direction?: 'column' | 'row'
    /** Space between children, from the spacing scale (`--weft-space-*`). */
    gap?: Gap
    /** Cross-axis alignment of the children. */
    align?: 'start' | 'center' | 'end' | 'stretch'
    /** Main-axis distribution of the children; `between` pushes them to the ends. */
    justify?: 'start' | 'center' | 'end' | 'between'
    /** Let a row wrap onto more lines when its children do not fit. */
    wrap?: boolean
    /** The element to render, for semantics: `ul`, `nav`, `form`. */
    as?: string
  }>(),
  { direction: 'column', gap: 4, align: 'stretch', justify: 'start', wrap: false, as: 'div' },
)
</script>

<template>
  <component
    :is="as"
    class="weft-stack"
    :data-direction="direction"
    :data-gap="gap"
    :data-align="align"
    :data-justify="justify"
    :data-wrap="wrap || undefined"
  >
    <slot />
  </component>
</template>
