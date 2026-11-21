<script setup vapor lang="ts">
import { useInset } from '../adaptive/context.ts'
import '../styles/base.css'
import '../styles/page.css'

defineProps<{
  /** The page's heading, rendered as its h1 (an h2 inside a WeftProvider marked `inset`). */
  title?: string
  /** A line under the title saying what the page is for. */
  description?: string
}>()
// A preview of a page, inside another, is not that page's main content.
const inset = useInset()
</script>

<template>
  <component :is="inset ? 'div' : 'main'" class="weft-page">
    <header v-if="title || description || $slots.header || $slots.actions" class="weft-page__header">
      <div class="weft-page__heading">
        <slot name="header">
          <component :is="inset ? 'h2' : 'h1'" v-if="title" class="weft-page__title">{{ title }}</component>
          <p v-if="description" class="weft-page__description">{{ description }}</p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="weft-page__actions"><slot name="actions" /></div>
    </header>
    <div class="weft-page__body"><slot /></div>
  </component>
</template>
