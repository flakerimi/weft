<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, FieldGroup, FormField, Input, InputNumber, PinInput, Slider, Textarea, WeftProvider } from '../../../src/index.ts'
import type { Input as InputKind } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ case: string; input?: InputKind; dir?: 'ltr' | 'rtl'; wheel?: boolean; autoresize?: boolean; maxrows?: number }>()
const note = ref('first')
const qty = ref<number | null>(2)
const price = ref<number | null>(0.1)
const volume = ref<number | number[]>(40)
const span = ref<number | number[]>([20, 60])
const code = ref('')
const completed = ref<string[]>([])
const changes = ref<(number | number[])[]>([])
defineExpose({ note, qty, price, volume, span, code, completed, changes })
void props
</script>

<template>
  <div v-if="$props.case === 'textarea'">
    <Textarea v-model="note" class="bound" :autoresize="autoresize" :maxrows="maxrows" placeholder="Notes" />
    <FormField label="Bio" hint="A line or two." class="in-field"><Textarea /></FormField>
  </div>
  <div v-else-if="$props.case === 'number'">
    <InputNumber v-model="qty" class="qty" :min="0" :max="10" :wheel="wheel" aria-label="Quantity" />
    <InputNumber v-model="price" class="price" :step="0.1" aria-label="Price" />
    <InputNumber class="free" aria-label="Free" />
    <FormField label="Seats" class="in-field"><InputNumber /></FormField>
  </div>
  <div v-else-if="$props.case === 'slider'" :dir="dir">
    <Slider v-model="volume" class="single" aria-label="Volume" @change="changes.push($event)" />
    <Slider v-model="span" class="range" :min="0" :max="100" :step="10" @change="changes.push($event)" />
    <Slider class="off" disabled :default-value="5" aria-label="Off" />
    <Slider class="tall" orientation="vertical" :default-value="25" aria-label="Level" />
    <FormField label="Brightness" class="in-field"><Slider name="brightness" /></FormField>
    <FormField label="Price" class="range-field"><Slider :default-value="[10, 90]" /></FormField>
    <FormField label="Stops" class="three-field"><Slider :default-value="[10, 50, 90]" /></FormField>
  </div>
  <div v-else-if="$props.case === 'pin'" :dir="dir">
    <PinInput v-model="code" class="code" :length="4" type="number" otp @complete="completed.push($event)" />
    <PinInput class="masked" :length="3" mask />
    <FormField label="Code" name="code" class="in-field"><PinInput :length="2" /></FormField>
  </div>
  <div v-else-if="$props.case === 'group'">
    <FieldGroup class="row" size="sm">
      <Input class="first" aria-label="Site" />
      <Button label="Go" size="sm" />
    </FieldGroup>
    <FieldGroup class="column" orientation="vertical">
      <Input aria-label="A" />
      <Input aria-label="B" size="lg" />
    </FieldGroup>
  </div>
  <WeftProvider v-else-if="$props.case === 'adaptive'" :input="input">
    <InputNumber class="n" aria-label="N" />
    <Slider class="s" aria-label="S" />
    <PinInput class="p" :length="2" />
  </WeftProvider>
  <WeftProvider v-else-if="$props.case === 'small'" :input="input">
    <Input class="i" size="sm" aria-label="I" />
    <InputNumber class="n" size="sm" aria-label="N" />
    <PinInput class="p" size="sm" :length="2" />
    <Slider class="s" size="sm" aria-label="S" />
    <Slider class="l" size="lg" aria-label="L" />
    <Textarea class="t" size="sm" aria-label="T" />
  </WeftProvider>
</template>
