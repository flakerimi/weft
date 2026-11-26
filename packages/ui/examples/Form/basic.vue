<script setup vapor lang="ts">
import { reactive, ref } from 'vue'
import { Button, Checkbox, Form, FormField, Input, InputNumber, type FieldSchema } from '@weft/ui'

const state = reactive({ email: '', age: null as number | null, terms: false })
// The same rules the server checks; weft gen writes this object from the Go types.
const schema: FieldSchema<keyof typeof state> = {
  email: { required: true, email: true },
  age: { min: 18, message: 'You must be 18 or over.' },
  terms: { required: true, message: 'Accept the terms to go on.' },
}
const sent = ref('')
</script>

<template>
  <Form :state="state" :schema="schema" @submit="sent = $event.email">
    <FormField label="Email" name="email"><Input v-model="state.email" type="email" autocomplete="email" /></FormField>
    <FormField label="Age" name="age"><InputNumber v-model="state.age" :min="0" /></FormField>
    <FormField name="terms"><Checkbox v-model="state.terms" label="I accept the terms" /></FormField>
    <Button type="submit" label="Create account" />
    <p v-if="sent">Welcome, {{ sent }}.</p>
  </Form>
</template>
