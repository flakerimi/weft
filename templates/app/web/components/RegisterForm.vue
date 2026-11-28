<script setup vapor lang="ts">
// An island: asks for a sign-up link with api.auth.register. The server
// answers the same whether the address has an account or not, and mails
// either a link that finishes signing up or one that resets the password,
// so the page says only that a mail may be on its way.
import { reactive, ref, useTemplateRef } from 'vue'
import type { FormApi } from '@weft/ui'
import { api, schema, type RegisterInput } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'

const state = reactive<RegisterInput>({ email: '' })
const problem = ref('')
const sent = ref(false)
const form = useTemplateRef<FormApi>('form')

async function register(values: RegisterInput) {
  problem.value = ''
  try {
    await api.auth.register(values)
    sent.value = true
  } catch (e) {
    const { fields, message } = formProblem(e)
    form.value?.setErrors(fields)
    problem.value = message
  }
}
</script>

<template>
  <Alert
    v-if="sent"
    tone="success"
    title="If that address can be used, an email is on its way; open the link in it within a day."
  />
  <Form v-else ref="form" :state="state" :schema="schema.RegisterInput" @submit="register">
    <Stack :gap="4">
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Email" name="email">
        <Input v-model="state.email" type="email" autocomplete="email" />
      </FormField>
      <Button type="submit" label="Send the link" />
      <Link to="/login">Sign in instead</Link>
    </Stack>
  </Form>
</template>
