<script setup vapor lang="ts">
import '../styles/base.css'
import '../styles/section.css'

defineProps<{
  /** The section's heading, rendered as an h2. */
  title?: string
  /** A line under the title. */
  description?: string
}>()
</script>

<template>
  <section class="weft-section">
    <header v-if="title || description || $slots.actions" class="weft-section__header">
      <div class="weft-section__heading">
        <h2 v-if="title" class="weft-section__title">{{ title }}</h2>
        <p v-if="description" class="weft-section__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="weft-section__actions"><slot name="actions" /></div>
    </header>
    <slot />
  </section>
</template>
