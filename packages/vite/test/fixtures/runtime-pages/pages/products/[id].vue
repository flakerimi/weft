<script lang="ts">
// Built once, as a shell, and served for every /products/<id>.
export const runtime = true
</script>

<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Counter from '../../components/Counter.vue'

defineProps<{ id: string }>()
</script>

<template>
  <main>
    <h1>Product [{{ id }}]</h1>
    <Island when="load" :component="Counter" :props="{ start: 1 }" />
  </main>
</template>
