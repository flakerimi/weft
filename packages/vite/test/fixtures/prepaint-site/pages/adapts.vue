<script setup vapor lang="ts">
import { Stepper, Tabs, Timeline } from '@weft/ui'
const steps = [{ title: 'Account' }, { title: 'Address' }, { title: 'Payment' }]
const tabs = [{ label: 'One', content: 'First' }, { label: 'Two', content: 'Second' }]
</script>

<template>
  <main>
    <Stepper id="st" :items="steps" />
    <Timeline id="tl" orientation="horizontal" :items="steps" />
    <Tabs id="tb" orientation="vertical" :items="tabs" aria-label="Sections" />
  </main>
</template>
