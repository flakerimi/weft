<script setup vapor lang="ts">
// A sign-in form in an island (form-guard.test.ts): before it hydrates a
// submit must send nothing, and once it is live a submit runs the handler,
// counted in the output, once, with what was typed before it was live.
import { reactive, ref } from 'vue'
import { Button, Form, FormField, Input } from '@weft/ui'

const state = reactive({ email: '', password: '' })
const sent = ref(0)
const got = ref('')
function send(values: { email: string; password: string }) {
  sent.value++
  got.value = `${values.email}|${values.password}`
}
</script>

<template>
  <Form :state="state" @submit="send">
    <FormField label="Email" name="email"><Input v-model="state.email" type="email" /></FormField>
    <FormField label="Password" name="password"><Input v-model="state.password" type="password" /></FormField>
    <Button type="submit" label="Sign in" />
    <output>{{ sent }}</output>
    <output class="got">{{ got }}</output>
  </Form>
</template>
