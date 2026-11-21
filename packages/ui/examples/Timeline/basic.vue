<script setup vapor lang="ts">
import { Timeline } from '@weft/ui'

const events = [
  { value: 'ordered', date: 'Mar 1, 09:12', title: 'Order placed', description: 'Paid by card ending 4242.', icon: 'shopping-cart' },
  { value: 'shipped', date: 'Mar 2, 16:40', title: 'Shipped', description: 'Handed to the courier in Prishtina.', icon: 'truck' },
  { value: 'delivered', date: 'Expected Mar 4', title: 'Delivered', icon: 'package-check' },
]
</script>

<template>
  <Timeline :items="events" model-value="shipped" />
</template>
