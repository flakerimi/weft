<script setup vapor lang="ts">
import { ref } from 'vue'
import { Accordion, type AccordionItem } from '@weft/ui'

const open = ref<string | string[] | null>(['billing'])
const items: AccordionItem[] = [
  { label: 'Account', value: 'account', icon: 'user', content: 'Name, email and password.' },
  { label: 'Billing', value: 'billing', icon: 'credit-card', content: 'Your plan and invoices.' },
  { label: 'Team', value: 'team', icon: 'users', disabled: true, content: 'Available on the Team plan.' },
]
</script>

<template>
  <Accordion v-model="open" type="multiple" :items="items" />
</template>
