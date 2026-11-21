<script setup vapor lang="ts">
import { ref } from 'vue'
import { Checkbox, CheckboxGroup, FormField, RadioGroup, Switch, WeftProvider } from '../../../src/index.ts'
import type { Input as InputKind } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ case: string; input?: InputKind; dir?: 'ltr' | 'rtl'; refuse?: boolean }>()
const ticked = ref<boolean | 'indeterminate'>(false)
const all = ref<boolean | 'indeterminate'>('indeterminate')
const fruits = ref<(string | number)[]>(['apple'])
const plan = ref<string | number | null>('pro')
const on = ref(false)
const busy = ref(true)
const changes = ref(0)
function setTicked(v: boolean | 'indeterminate') {
  if (!props.refuse) ticked.value = v
}
defineExpose({ ticked, all, fruits, plan, on, busy, changes })
const plans = [
  { value: 'free', label: 'Free' },
  { value: 'pro', label: 'Pro', description: 'For teams.' },
  { value: 'old', label: 'Legacy', disabled: true },
  { value: 'max', label: 'Max' },
]
</script>

<template>
  <div v-if="$props.case === 'checkbox'">
    <Checkbox class="bound" :model-value="ticked" label="Accept" description="You can change it later." @update:model-value="setTicked" />
    <Checkbox class="free" label="Free" />
    <Checkbox v-model="all" class="mixed" label="All" />
    <Checkbox class="off" label="Off" disabled />
    <span class="echo">{{ ticked }}</span>
  </div>
  <div v-else-if="$props.case === 'group'">
    <CheckboxGroup v-model="fruits" class="items" legend="Fruit" name="fruit" :items="['apple', 'pear', { value: 'fig', label: 'Fig', disabled: true }]" />
    <CheckboxGroup v-model="fruits" class="slotted" legend="Again">
      <Checkbox value="apple" label="Apple" />
      <Checkbox value="pear" label="Pear" />
    </CheckboxGroup>
    <FormField label="Toppings" hint="Pick any." error="Pick one." class="in-field">
      <CheckboxGroup :items="['cheese', 'ham']" />
    </FormField>
  </div>
  <div v-else-if="$props.case === 'radio'" :dir="dir">
    <RadioGroup v-model="plan" class="plans" legend="Plan" :items="plans" />
    <RadioGroup class="none" legend="Size" :items="['s', 'm', 'l']" />
    <FormField label="Tier" error="Choose a tier." required class="in-field">
      <RadioGroup :items="['a', 'b']" />
    </FormField>
    <span class="echo">{{ plan }}</span>
  </div>
  <div v-else-if="$props.case === 'switch'">
    <Switch v-model="on" class="bound" label="Wi-Fi" description="Join known networks." @change="changes++" />
    <Switch v-model="busy" class="busy" label="Sync" loading />
    <FormField label="Alerts" class="in-field"><Switch /></FormField>
  </div>
  <WeftProvider v-else-if="$props.case === 'adaptive'" :input="input">
    <Checkbox class="c" label="Tick" />
    <Switch class="s" label="On" />
    <RadioGroup class="r" :items="['a']" />
  </WeftProvider>
</template>
