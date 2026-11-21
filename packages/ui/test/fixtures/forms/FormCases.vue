<script setup vapor lang="ts">
import { reactive, ref, useTemplateRef } from 'vue'
import { Button, Checkbox, Form, FormField, Input, InputNumber, RadioGroup, type FieldSchema } from '../../../src/index.ts'

defineProps<{ disabled?: boolean }>()
const state = reactive({ email: '', age: null as number | null, terms: false, plan: null as string | number | null, nick: '' })
const schema: FieldSchema<keyof typeof state> = {
  email: { required: true, email: true },
  age: { min: 18, message: 'You must be 18 or over.' },
  terms: { required: true },
  plan: { required: true },
}
const submitted = ref<(typeof state)[]>([])
const failed = ref<Record<string, string>[]>([])
const form = useTemplateRef<{ setErrors(e: Record<string, string>): void; validate(): boolean; clear(): void; errors: Record<string, string> }>('form')
defineExpose({
  state,
  submitted,
  failed,
  setErrors: (e: Record<string, string>) => form.value?.setErrors(e),
  validate: () => form.value?.validate(),
  clear: () => form.value?.clear(),
  errors: () => form.value?.errors,
})
</script>

<template>
  <Form ref="form" :state="state" :schema="schema" :disabled="disabled" @submit="submitted.push($event)" @error="failed.push($event)">
    <FormField label="Email" name="email" class="email"><Input v-model="state.email" type="email" /></FormField>
    <FormField label="Age" name="age" class="age"><InputNumber v-model="state.age" /></FormField>
    <FormField name="terms" class="terms"><Checkbox v-model="state.terms" label="I accept the terms" /></FormField>
    <FormField label="Plan" name="plan" class="plan"><RadioGroup v-model="state.plan" :items="['free', 'pro']" /></FormField>
    <FormField label="Nickname" name="nick" class="nick"><Input v-model="state.nick" /></FormField>
    <Button type="submit" label="Create account" />
  </Form>
</template>
