<script setup vapor lang="ts">
// An island: signs the owner in with api.auth.login, then goes to the page
// that sent the visitor here (?next=), or to the front page.
import { onMounted, reactive, ref, useTemplateRef } from 'vue'
import { api, schema, type LoginInput } from '../.weft-types/api.ts'
import { formProblem, nextPath, useSession } from '../session.ts'

const state = reactive<LoginInput>({ email: '', password: '' })
const problem = ref('')
const form = useTemplateRef<{ setErrors(errors: Record<string, string>): void }>('form')
// Form's server-rendered markup is a plain <form> that the browser submits as
// a GET to this page until the island hydrates, which would put the password
// in the address; so the form is drawn only once mounted.
const mounted = ref(false)

const next = () => nextPath(new URLSearchParams(location.search).get('next'))

// Someone already signed in goes on at once.
onMounted(async () => {
  mounted.value = true
  if (await useSession().ready) location.assign(next())
})

async function signIn(values: LoginInput) {
  problem.value = ''
  try {
    await api.auth.login(values)
    location.assign(next())
  } catch (e) {
    const { fields, message } = formProblem(e)
    form.value?.setErrors(fields)
    problem.value = message
  }
}
</script>

<template>
  <Skeleton v-if="!mounted" height="12rem" />
  <Form v-else ref="form" :state="state" :schema="schema.LoginInput" @submit="signIn">
    <Stack :gap="4">
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Email" name="email">
        <Input v-model="state.email" type="email" autocomplete="username" />
      </FormField>
      <FormField label="Password" name="password">
        <Input v-model="state.password" type="password" autocomplete="current-password" />
      </FormField>
      <Button type="submit" label="Sign in" />
    </Stack>
  </Form>
</template>
