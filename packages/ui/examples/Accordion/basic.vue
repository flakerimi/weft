<script setup vapor lang="ts">
import { Accordion, type AccordionItem } from '@weft/ui'

const items: AccordionItem[] = [
  { label: 'How long does shipping take?', content: 'Two to four working days within the EU.' },
  { label: 'Can I return an order?', content: 'Yes, within thirty days, in its original packaging.' },
  { label: 'Do you ship abroad?', content: 'To most countries; the checkout shows the cost.' },
]
</script>

<template>
  <Accordion :items="items" />
</template>
