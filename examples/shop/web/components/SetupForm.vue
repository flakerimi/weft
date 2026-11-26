<script setup vapor lang="ts">
// An island: sets the owner's password with the one-time token from the
// link the app logs at its first start (/setup#token=…), which signs the
// owner in. The token is in the address's fragment, which never reaches a
// server log or a Referer header.
import { onMounted, reactive, ref, useTemplateRef } from 'vue'
import { api, schema, type SetupInput } from '../.weft-types/api.ts'
import { formProblem } from '../session.ts'

const state = reactive<SetupInput>({ token: '', password: '' })
const problem = ref('')
const noToken = ref(false)
const form = useTemplateRef<{ setErrors(errors: Record<string, string>): void }>('form')
// Form's server-rendered markup is a plain <form> that the browser submits as
// a GET to this page until the island hydrates, which would put the password
// in the address; so the form is drawn only once mounted.
const mounted = ref(false)

onMounted(() => {
  mounted.value = true
  state.token = new URLSearchParams(location.hash.slice(1)).get('token') ?? ''
  noToken.value = state.token === ''
})

async function setUp(values: SetupInput) {
  problem.value = ''
  try {
    await api.auth.setup(values)
    location.assign('/products')
  } catch (e) {
    const { fields, message } = formProblem(e)
    // The token has no field of its own: its message goes at the top.
    const { token, ...rest } = fields
    form.value?.setErrors(rest)
    problem.value = token ?? message
  }
}
</script>

<template>
  <Skeleton v-if="!mounted" height="8rem" />
  <Alert
    v-else-if="noToken"
    tone="warning"
    title="This page needs the setup link."
    description="Open the link the app wrote to its log when it started, which ends in #token=…."
  />
  <Form v-else ref="form" :state="state" :schema="schema.SetupInput" @submit="setUp">
    <Stack :gap="4">
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Password" name="password" description="At least 8 characters.">
        <Input v-model="state.password" type="password" autocomplete="new-password" />
      </FormField>
      <Button type="submit" label="Set up the owner" />
    </Stack>
  </Form>
</template>
