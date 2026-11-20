<script setup vapor lang="ts">
import type { Gap } from './types.ts'

withDefaults(
  defineProps<{
    direction?: 'column' | 'row'
    gap?: Gap
    align?: 'start' | 'center' | 'end' | 'stretch'
    justify?: 'start' | 'center' | 'end' | 'between'
    wrap?: boolean
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
