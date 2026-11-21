<script setup vapor lang="ts">
import { ref } from 'vue'
import Tabs from '../../../src/navigation/Tabs.vue'
import MountSpy from './MountSpy.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

defineProps<{
  activation?: 'automatic' | 'manual'
  orientation?: 'horizontal' | 'vertical'
  keepAlive?: boolean
  hash?: boolean
  dir?: 'ltr' | 'rtl'
  input?: Input
}>()
const tab = ref<string | number>('account')
const mounts: string[] = []
const items = [
  { label: 'Account', value: 'account', icon: 'user', content: 'Your account.' },
  { label: 'Password', value: 'password', slot: 'password' },
  { label: 'Billing', value: 'billing', disabled: true, content: 'Billing.' },
  { label: 'Team', value: 'team', badge: 3, content: 'Your team.' },
]
defineExpose({ tab, mounts })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <Tabs
        v-model="tab"
        :items="items"
        :activation="activation"
        :orientation="orientation"
        :keep-alive="keepAlive"
        :hash="hash"
        aria-label="Settings"
      >
        <template #password><MountSpy name="password" :log="mounts" /></template>
      </Tabs>
    </div>
  </WeftProvider>
</template>
