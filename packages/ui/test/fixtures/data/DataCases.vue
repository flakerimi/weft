<script setup vapor lang="ts">
import StatCard from '../../../src/data/StatCard.vue'
import Timeline from '../../../src/data/Timeline.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'

withDefaults(
  defineProps<{
    delta?: number
    invert?: boolean
    loading?: boolean
    value?: string | number
    locale?: string
    current?: string | number
    orientation?: 'vertical' | 'horizontal'
    reverse?: boolean
  }>(),
  { delta: undefined, value: 12480, current: undefined },
)
const events = [
  { date: 'Mar 1', title: 'Ordered', description: 'Paid by card', value: 'ordered', icon: 'shopping-cart' },
  { date: 'Mar 2', title: 'Shipped', value: 'shipped' },
  { date: 'Mar 4', title: 'Delivered', value: 'delivered' },
]
</script>

<template>
  <WeftProvider :locale="locale">
    <StatCard label="Revenue" :value="value" :delta="delta" :invert="invert" description="vs last month" :loading="loading" icon="banknote">
      <template #sparkline><svg class="spark" viewBox="0 0 10 4"><path d="M0 4 L10 0" /></svg></template>
    </StatCard>
    <Timeline :items="events" :model-value="current" :orientation="orientation" :reverse="reverse">
      <template #description="{ item }"><em class="desc">{{ item.description ?? 'none' }}</em></template>
    </Timeline>
  </WeftProvider>
</template>
