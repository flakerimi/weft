<script setup vapor lang="ts">
withDefaults(defineProps<{ padding?: 0 | 2 | 3 | 4 | 6; as?: string }>(), { padding: 4, as: 'div' })
</script>

<template>
  <component :is="as" class="weft-card" :data-padding="padding">
    <div v-if="$slots.header || $slots.actions" class="weft-card__header">
      <div class="weft-card__title"><slot name="header" /></div>
      <div v-if="$slots.actions" class="weft-card__actions"><slot name="actions" /></div>
    </div>
    <div class="weft-card__body"><slot /></div>
    <div v-if="$slots.footer" class="weft-card__footer"><slot name="footer" /></div>
  </component>
</template>
