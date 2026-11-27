<script setup vapor lang="ts">
import { Footer, Pagination, Stepper, Table, Tabs, Timeline } from '@weft/ui'
const steps = [{ title: 'Account' }, { title: 'Address' }, { title: 'Payment' }]
const tabs = [{ label: 'One', content: 'First' }, { label: 'Two', content: 'Second' }]
const columns = [{ key: 'name', label: 'Name' }]
const rows = [{ id: 1, name: 'Ada' }]
</script>

<template>
  <main>
    <Stepper id="st" orientation="vertical" :items="steps" />
    <Timeline id="tl" :items="steps" />
    <Tabs id="tb" :items="tabs" aria-label="Sections" />
    <Pagination :total="200" :default-page="10" />
    <Table caption="People" :columns="columns" :rows="rows" />
  </main>
  <Footer>© 2026</Footer>
</template>
