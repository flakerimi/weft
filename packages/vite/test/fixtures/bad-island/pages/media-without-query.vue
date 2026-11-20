<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Counter from '../../site/components/Counter.vue'
</script>

<template>
  <Island when="media" :component="Counter" :props="{ start: 1 }" />
</template>
