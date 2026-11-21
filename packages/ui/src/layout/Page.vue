<script setup vapor lang="ts">
import '../styles/base.css'
import '../styles/page.css'

defineProps<{
  /** The page's heading, rendered as its h1. */
  title?: string
  /** A line under the title saying what the page is for. */
  description?: string
}>()
</script>

<template>
  <main class="weft-page">
    <header v-if="title || description || $slots.header || $slots.actions" class="weft-page__header">
      <div class="weft-page__heading">
        <slot name="header">
          <h1 v-if="title" class="weft-page__title">{{ title }}</h1>
          <p v-if="description" class="weft-page__description">{{ description }}</p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="weft-page__actions"><slot name="actions" /></div>
    </header>
    <div class="weft-page__body"><slot /></div>
  </main>
</template>
