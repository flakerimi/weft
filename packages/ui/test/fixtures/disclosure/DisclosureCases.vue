<script setup vapor lang="ts">
import { ref } from 'vue'
import Accordion from '../../../src/disclosure/Accordion.vue'
import Collapsible from '../../../src/disclosure/Collapsible.vue'

const props = withDefaults(defineProps<{ type?: 'single' | 'multiple'; collapsible?: boolean; dir?: 'ltr' | 'rtl' }>(), { collapsible: true })
const open = ref(false)
const section = ref<string | string[] | null>(props.type === 'multiple' ? [] : null)
const items = [
  { label: 'Shipping', value: 'shipping', content: 'Ships in two days.' },
  { label: 'Returns', value: 'returns', icon: 'undo', content: 'Thirty days to return.' },
  { label: 'Warranty', value: 'warranty', disabled: true, content: 'One year.' },
  { label: 'Support', value: 'support', slot: 'support' },
]
defineExpose({ open, section })
</script>

<template>
  <div :dir="dir">
    <Collapsible v-model:open="open" class="slotted">
      <button class="trigger">Details</button>
      <template #content><p class="inside">Hidden details</p></template>
    </Collapsible>
    <Collapsible label="More" class="labelled" disabled>
      <template #content>Labelled content</template>
    </Collapsible>
    <Accordion v-model="section" :items="items" :type="type" :collapsible="collapsible">
      <template #support="{ item }"><span class="support">Mail us about {{ item.label }}</span></template>
    </Accordion>
  </div>
</template>
