<script setup vapor lang="ts">
import { reactive, ref, useTemplateRef } from 'vue'
import { Button, Form, FormField, Input } from '@weft/ui'

const state = reactive({ username: '' })
const form = useTemplateRef<{ setErrors(errors: Record<string, string>): void }>('form')
const busy = ref(false)

// A stand-in for the server's answer: the name is taken.
async function save(values: { username: string }) {
  busy.value = true
  await new Promise((r) => setTimeout(r, 600))
  busy.value = false
  form.value?.setErrors({ username: `"${values.username}" is taken; try another.` })
}
</script>

<template>
  <Form ref="form" :state="state" :schema="{ username: { required: true, min: 3 } }" @submit="save">
    <FormField label="Username" name="username"><Input v-model="state.username" /></FormField>
    <Button type="submit" label="Claim it" :loading="busy" />
  </Form>
</template>
