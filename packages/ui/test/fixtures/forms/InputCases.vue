<script setup vapor lang="ts">
import { ref, useTemplateRef } from 'vue'
import { Button, FormField, Input, WeftProvider } from '../../../src/index.ts'
import type { Input as InputKind } from '../../../src/adaptive/detect.ts'
import type { Size } from '../../../src/types.ts'

const props = defineProps<{ case: string; input?: InputKind; size?: Size; error?: string | boolean; initial?: string }>()
const text = ref<string | number | null>(props.initial ?? 'hello')
const trimmed = ref('')
const amount = ref<string | number | null>(null)
const lazy = ref('')
const field = useTemplateRef<{ focus(): void }>('field')
defineExpose({ text, trimmed, amount, lazy, focusField: () => field.value?.focus() })
</script>

<template>
  <div v-if="$props.case === 'model'">
    <Input v-model="text" class="bound" />
    <Input class="free" />
    <Input v-model.trim="trimmed" class="trim" />
    <Input v-model.number="amount" class="number" />
    <Input v-model.lazy="lazy" class="lazy" />
    <Input ref="field" class="exposed" />
    <span class="echo">{{ text }}</span>
  </div>
  <div v-else-if="$props.case === 'ends'">
    <Input class="slots" placeholder="Search">
      <template #leading><i class="lead">L</i></template>
      <template #trailing><button class="clear" type="button">Clear</button></template>
    </Input>
  </div>
  <FormField
    v-else-if="$props.case === 'field'"
    label="Email"
    hint="We never share it."
    :error="error"
    required
    name="email"
  >
    <Input type="email" aria-describedby="extra" />
  </FormField>
  <FormField v-else-if="$props.case === 'bare-field'" label="Name">
    <Input />
  </FormField>
  <FormField v-else-if="$props.case === 'error-slot'" label="Code" :error="true">
    <Input />
    <template #error><b class="why">Ask your admin.</b></template>
  </FormField>
  <WeftProvider v-else-if="$props.case === 'adaptive'" :input="input">
    <Input class="plain" :size="size" />
    <Button class="button" :size="size" label="Go" />
  </WeftProvider>
</template>
