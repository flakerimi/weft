<script setup vapor lang="ts">
import { ref } from 'vue'
import NavigationMenu from '../../../src/navigation/NavigationMenu.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { NavItem } from '../../../src/navigation/nav.ts'

const props = defineProps<{ input?: Input; orientation?: 'horizontal' | 'vertical'; collapsed?: boolean; dir?: 'ltr' | 'rtl' }>()
const input = ref<Input | undefined>(props.input)
const chosen: string[] = []
const items = ref<NavItem[][]>([
  [
    { label: 'Home', to: '/', icon: 'house' },
    { label: 'Docs', to: '/docs', active: true },
    {
      label: 'Components',
      icon: 'box',
      children: [
        { label: 'Button', to: '/components/button', description: 'A button' },
        { label: 'Tabs', to: '/components/tabs' },
      ],
    },
    { label: 'Old', to: '/old', disabled: true },
  ],
  [
    { type: 'label', label: 'More' },
    { label: 'Inbox', to: '/inbox', badge: 4 },
    { label: 'Sign out', onSelect: () => chosen.push('sign out') },
  ],
])
defineExpose({ input, chosen, items })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <NavigationMenu :items="items" :orientation="orientation" :collapsed="collapsed" label="Site" @select="(item) => chosen.push(item.label!)" />
    </div>
  </WeftProvider>
</template>
