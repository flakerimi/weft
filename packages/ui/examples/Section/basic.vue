<script setup vapor lang="ts">
import { Button, Section } from '@weft/ui'
</script>

<template>
  <Section title="Billing" description="How and when you pay.">
    <template #actions><Button label="Edit" size="sm" variant="outline" tone="neutral" /></template>
    <p>Visa ending 4242, renews on 1 December.</p>
  </Section>
</template>
