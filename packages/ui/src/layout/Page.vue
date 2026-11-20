<script setup vapor lang="ts">
defineProps<{ title?: string; description?: string }>()
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
