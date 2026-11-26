<script setup vapor lang="ts">
// An island: sets the owner's password with the one-time token from the
// link the app logs at its first start (/setup#token=…), which signs the
// owner in. The token is in the address's fragment, which never reaches a
// server log or a Referer header; once read it is taken out of the address
// too, so it is not in the address bar, the history or a copied link. The
// link pasted again into the same tab changes only the fragment, so the page
// reads it again then.
import { onMounted, onUnmounted, reactive, ref, useTemplateRef } from 'vue'
import type { FormApi } from '@weft/ui'
import { api, schema, type SetupInput } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'

const state = reactive<SetupInput>({ token: '', password: '' })
const problem = ref('')
const noToken = ref(false)
const form = useTemplateRef<FormApi>('form')

function readToken() {
  const token = new URLSearchParams(location.hash.slice(1)).get('token') ?? ''
  if (location.hash) history.replaceState(history.state, '', location.pathname + location.search)
  // A fragment with no token, pasted later, leaves a token already read.
  if (token !== '' || state.token === '') state.token = token
  noToken.value = state.token === ''
}

onMounted(() => {
  readToken()
  window.addEventListener('hashchange', readToken)
})
onUnmounted(() => window.removeEventListener('hashchange', readToken))

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
  <Alert
    v-if="noToken"
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
