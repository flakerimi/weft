<script setup vapor lang="ts">
import { Page, Section } from '@weft/ui'
</script>

<template>
  <Page title="Customers" description="Everyone with an active plan.">
    <template #actions><button type="button">New customer</button></template>
    <Section title="Recent">
      <p>No customers yet.</p>
    </Section>
  </Page>
</template>
