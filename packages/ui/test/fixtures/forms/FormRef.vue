<script setup vapor lang="ts">
// A Form reached through its template ref, typed with FormApi as an app
// types it, and the ref its type is inferred from: vue-tsc checks that what
// the component exposes is a FormApi.
import { reactive, useTemplateRef } from 'vue'
import { Form, FormField, Input, type FormApi } from '../../../src/index.ts'

const state = reactive({ email: '' })
const form = useTemplateRef<FormApi>('form')
const other = useTemplateRef('other')
// Assignable only while the exposed instance has every member of FormApi, typed alike.
const check = (): FormApi | null => other.value
defineExpose({ form, check })
</script>

<template>
  <Form ref="form" :state="state" :schema="{ email: { required: true } }">
    <FormField label="Email" name="email" class="email"><Input v-model="state.email" /></FormField>
  </Form>
  <Form ref="other" :state="{}" />
</template>
