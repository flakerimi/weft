<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Counter from '../components/Counter.vue'
</script>

<template>
  <Island when="media" media="(min-width: 40rem)" :component="Counter" :props="{ start: 1, note: '</script><!--&' }" />
</template>
