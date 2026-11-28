<script setup vapor lang="ts">
// An island: signs a user in with api.auth.login, then goes to the page that
// sent the visitor here (?next=), when that is a path on this site, or to the
// front page. It links to /forgot, and to /register when the app takes
// sign-ups (AUTH_REGISTRATION=open).
import { onMounted, reactive, ref, useTemplateRef } from 'vue'
import { nextPath, useSession, type FormApi } from '@weft/ui'
import { api, schema, type LoginInput } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'

const state = reactive<LoginInput>({ email: '', password: '' })
const problem = ref('')
const signUps = ref(false)
const form = useTemplateRef<FormApi>('form')

const next = () => nextPath(new URLSearchParams(location.search).get('next'), location.origin)

// Someone already signed in goes on at once.
onMounted(async () => {
  if (await useSession().ready) location.assign(next())
})
onMounted(async () => {
  try {
    signUps.value = (await api.auth.settings()).registration === 'open'
  } catch {
    // Without the setting, no link to sign up: the form works as it is.
  }
})

async function signIn(values: LoginInput) {
  problem.value = ''
  try {
    await api.auth.login(values)
    location.assign(next())
  } catch (e) {
    // A 401 here is a wrong password, not a session that ended.
    const { fields, message } = formProblem(e)
    form.value?.setErrors(fields)
    problem.value = message
  }
}
</script>

<template>
  <Form ref="form" :state="state" :schema="schema.LoginInput" @submit="signIn">
    <Stack :gap="4">
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Email" name="email">
        <Input v-model="state.email" type="email" autocomplete="username" />
      </FormField>
      <FormField label="Password" name="password">
        <Input v-model="state.password" type="password" autocomplete="current-password" />
      </FormField>
      <Button type="submit" label="Sign in" />
      <Stack direction="row" :gap="4" wrap>
        <Link to="/forgot">Forgot your password?</Link>
        <Link v-if="signUps" to="/register">Sign up</Link>
      </Stack>
    </Stack>
  </Form>
</template>
