<script setup vapor lang="ts">
import { ref, useTemplateRef } from 'vue'
import Stepper from '../../../src/navigation/Stepper.vue'

withDefaults(defineProps<{ linear?: boolean; orientation?: 'horizontal' | 'vertical'; dir?: 'ltr' | 'rtl' }>(), { linear: true })
const step = ref<string | number>('address')
const stepper = useTemplateRef<{ next: () => void; prev: () => void; hasNext: boolean; hasPrev: boolean }>('stepper')
const items = [
  { title: 'Address', value: 'address', description: 'Where to ship', icon: 'house', content: 'Address form' },
  { title: 'Shipping', value: 'shipping', slot: 'shipping' },
  { title: 'Payment', value: 'payment', content: 'Payment form' },
  { title: 'Review', value: 'review', disabled: true, content: 'Review' },
]
defineExpose({ step, stepper })
</script>

<template>
  <div :dir="dir">
    <Stepper ref="stepper" v-model="step" :items="items" :linear="linear" :orientation="orientation">
      <template #shipping="{ item }"><p class="shipping">Pick {{ item.title }}</p></template>
    </Stepper>
  </div>
</template>
