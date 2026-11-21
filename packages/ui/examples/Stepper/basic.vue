<script setup vapor lang="ts">
import { Stepper, type StepperItem } from '@weft/ui'

const items: StepperItem[] = [
  { title: 'Address', description: 'Where to ship', content: 'The address form.' },
  { title: 'Shipping', description: 'How fast', content: 'Standard or express.' },
  { title: 'Payment', description: 'Card or invoice', content: 'The payment form.' },
]
</script>

<template>
  <Stepper :items="items" />
</template>
