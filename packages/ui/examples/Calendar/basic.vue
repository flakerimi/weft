<script setup vapor lang="ts">
import { ref } from 'vue'
import { Calendar } from '@weft/ui'

// An ISO date string; on touch the dates are 44 px and a swipe changes the month.
const day = ref('2026-11-21')
</script>

<template>
  <Calendar v-model="day" aria-label="Delivery day" />
</template>
