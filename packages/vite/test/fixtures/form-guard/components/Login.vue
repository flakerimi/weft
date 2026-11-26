<script setup vapor lang="ts">
// A sign-in form in an island (form-guard.test.ts): before it hydrates a
// submit must send nothing, and once it is live a submit runs the handler,
// counted in the output, once.
import { reactive, ref } from 'vue'
import { Button, Form, FormField, Input } from '@weft/ui'

const state = reactive({ email: '', password: '' })
const sent = ref(0)
</script>

<template>
  <Form :state="state" @submit="sent++">
    <FormField label="Email" name="email"><Input v-model="state.email" type="email" /></FormField>
    <FormField label="Password" name="password"><Input v-model="state.password" type="password" /></FormField>
    <Button type="submit" label="Sign in" />
    <output>{{ sent }}</output>
  </Form>
</template>
