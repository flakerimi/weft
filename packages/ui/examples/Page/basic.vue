<script setup vapor lang="ts">
import { Button, Page, Section } from '@weft/ui'
</script>

<template>
  <Page title="Customers" description="Everyone with an active plan.">
    <template #actions><Button label="New customer" icon="plus" /></template>
    <Section title="Recent">
      <p>No customers yet.</p>
    </Section>
  </Page>
</template>
