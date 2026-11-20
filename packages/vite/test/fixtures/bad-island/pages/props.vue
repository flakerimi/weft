<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Counter from '../../site/components/Counter.vue'
// The test passes the value the island gets as its `extra` prop.
defineProps<{ value?: unknown }>()
</script>

<template>
  <Island when="load" :component="Counter" :props="{ start: 1, extra: value }" />
</template>
