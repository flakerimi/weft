<script setup vapor lang="ts">
import { ref } from 'vue'
import { Input, Tabs, type TabsItem } from '@weft/ui'

const tab = ref<string | number>('profile')
const items: TabsItem[] = [
  { label: 'Profile', value: 'profile', slot: 'profile' },
  { label: 'Notifications', value: 'notifications', content: 'Email and push settings.' },
  { label: 'Danger zone', value: 'danger', content: 'Delete the account.' },
]
</script>

<template>
  <Tabs v-model="tab" :items="items" orientation="vertical" activation="manual" variant="link" keep-alive hash aria-label="Account">
    <template #profile><Input placeholder="Display name" /></template>
  </Tabs>
</template>
