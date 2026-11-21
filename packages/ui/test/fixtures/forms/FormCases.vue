<script setup vapor lang="ts">
import { reactive, ref, useTemplateRef } from 'vue'
import {
  Button, Checkbox, CheckboxGroup, FieldGroup, Form, FormField, Input, InputNumber, RadioGroup, type FieldSchema,
} from '../../../src/index.ts'

const props = defineProps<{ disabled?: boolean; loading?: boolean; slow?: boolean }>()
const state = reactive({
  email: '',
  age: null as number | null,
  terms: false,
  plan: null as string | number | null,
  nick: '',
  fruit: ['apple'] as (string | number)[],
  loose: '',
})
const schema: FieldSchema<keyof typeof state> = {
  email: { required: true, email: true },
  age: { min: 18, message: 'You must be 18 or over.' },
  terms: { required: true },
  plan: { required: true },
}
const submitted = ref<(typeof state)[]>([])
const failed = ref<Record<string, string>[]>([])
// With `slow`, the handler returns a promise the test settles with finish().
let finish: () => void = () => {}
function onSubmit(values: typeof state) {
  submitted.value.push(values)
  if (props.slow) return new Promise<void>((resolve) => (finish = resolve))
}
const form = useTemplateRef<{ setErrors(e: Record<string, string>): void; validate(): boolean; clear(): void; errors: Record<string, string> }>('form')
defineExpose({
  state,
  submitted,
  failed,
  finish: () => finish(),
  setErrors: (e: Record<string, string>) => form.value?.setErrors(e),
  validate: () => form.value?.validate(),
  clear: () => form.value?.clear(),
  errors: () => form.value?.errors,
})
</script>

<template>
  <Form ref="form" :state="state" :schema="schema" :disabled="disabled" :loading="loading" @submit="onSubmit" @error="failed.push($event)">
    <FormField label="Email" name="email" class="email"><Input v-model="state.email" type="email" /></FormField>
    <FormField label="Age" name="age" class="age"><InputNumber v-model="state.age" /></FormField>
    <FormField name="terms" class="terms"><Checkbox v-model="state.terms" label="I accept the terms" /></FormField>
    <FormField label="Plan" name="plan" class="plan"><RadioGroup v-model="state.plan" :items="['free', 'pro']" /></FormField>
    <FormField label="Nickname" name="nick" class="nick"><Input v-model="state.nick" /></FormField>
    <FormField label="Fruit" name="fruit" class="fruit"><CheckboxGroup v-model="state.fruit" :items="['apple', 'pear']" /></FormField>
    <Input v-model="state.loose" class="bare" aria-label="Loose" />
    <FieldGroup class="joined"><Input aria-label="Site" /><Button label="Check" /></FieldGroup>
    <Button type="submit" label="Create account" />
  </Form>
</template>
