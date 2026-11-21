<script setup vapor lang="ts">
import Toaster from '../../../src/overlays/Toaster.vue'

defineProps<{ max?: number }>()
</script>

<template>
  <button class="elsewhere">elsewhere</button>
  <Toaster :max="max ?? 5" />
</template>
