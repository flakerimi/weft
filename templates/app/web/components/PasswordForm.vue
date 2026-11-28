<script setup vapor lang="ts">
// An island: sets a password with the token a mailed link carries
// (/password#token=…), for an invitation, a sign-up or a reset, then signs
// in and goes to the front page. The token is in the address's fragment,
// which never reaches a server log or a Referer header; once read it is
// taken out of the address too. api.auth.token says what the link is for
// before the form is shown, so a link that no longer works says so at once.
import { onMounted, onUnmounted, reactive, ref, useTemplateRef } from 'vue'
import { useSession, type FormApi } from '@weft/ui'
import { api, schema, type PasswordInput, type TokenInfo } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'

const state = reactive<PasswordInput>({ token: '', password: '' })
const info = ref<TokenInfo | null>(null)
const problem = ref('')
const noToken = ref(false)
const form = useTemplateRef<FormApi>('form')

async function readToken() {
  const token = new URLSearchParams(location.hash.slice(1)).get('token') ?? ''
  if (location.hash) history.replaceState(history.state, '', location.pathname + location.search)
  // A fragment with no token, pasted later, leaves a token already read.
  if (token === '' && state.token !== '') return
  state.token = token
  noToken.value = token === ''
  info.value = null
  problem.value = ''
  if (noToken.value) return
  try {
    info.value = await api.auth.token({ token })
  } catch (e) {
    problem.value = formProblem(e).message
  }
}

onMounted(() => {
  void readToken()
  window.addEventListener('hashchange', readToken)
})
onUnmounted(() => window.removeEventListener('hashchange', readToken))

/** What the link is for, in the page's words. */
function title(i: TokenInfo): string {
  if (i.purpose === 'invite') return `Choose a password for an invitation to ${location.host}, as ${i.role}`
  if (i.purpose === 'register') return 'Finish signing up'
  return 'Choose a new password'
}

async function setPassword(values: PasswordInput) {
  problem.value = ''
  try {
    await api.auth.password(values)
    await useSession().refresh()
    location.assign('/')
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
    title="This page needs the link from your email."
    description="Open the link in the mail, which ends in #token=…, or ask for a new one from the sign-in page."
  />
  <Alert v-else-if="!info && problem" tone="danger" :title="problem">
    <template #actions><Link to="/forgot">Ask for a new link</Link></template>
  </Alert>
  <Spinner v-else-if="!info" label="Reading the link" />
  <Form v-else ref="form" :state="state" :schema="schema.PasswordInput" @submit="setPassword">
    <Stack :gap="4">
      <Alert tone="primary" :title="title(info)" :description="info.email" />
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Password" name="password" description="At least 8 characters.">
        <Input v-model="state.password" type="password" autocomplete="new-password" />
      </FormField>
      <Button type="submit" label="Set the password and sign in" />
    </Stack>
  </Form>
</template>
