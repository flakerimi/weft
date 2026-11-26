<script setup vapor lang="ts">
// Every form component in one island: server-rendered, then hydrated in
// place (forms-island.test.ts) and checked for mismatches. Its submit
// handler records the values (keep-typed.test.ts).
import { reactive } from 'vue'
import {
  Button, Checkbox, CheckboxGroup, FieldGroup, Form, FormField, Input, InputNumber,
  PinInput, RadioGroup, Slider, Switch, Textarea, type FieldSchema,
} from '@weft/ui'
import { sent } from '../sent.ts'

const state = reactive({
  email: '',
  bio: 'Hello',
  seats: 2 as number | null,
  terms: false,
  fruit: ['apple'] as (string | number)[],
  plan: 'pro' as string | number | null,
  alerts: true,
  volume: 30 as number | number[],
  code: '',
  site: '',
})
const schema: FieldSchema<keyof typeof state> = { email: { required: true, email: true }, terms: { required: true } }
</script>

<template>
  <Form :state="state" :schema="schema" @submit="(values) => sent.push(values)">
    <FormField label="Email" name="email" hint="Work address."><Input v-model="state.email" type="email" /></FormField>
    <FormField label="Bio" name="bio"><Textarea v-model="state.bio" autoresize /></FormField>
    <FormField label="Seats" name="seats"><InputNumber v-model="state.seats" :min="1" /></FormField>
    <FormField name="terms"><Checkbox v-model="state.terms" label="I accept" /></FormField>
    <FormField label="Fruit" name="fruit"><CheckboxGroup v-model="state.fruit" :items="['apple', 'pear']" /></FormField>
    <FormField label="Plan" name="plan"><RadioGroup v-model="state.plan" :items="['free', 'pro']" /></FormField>
    <Switch v-model="state.alerts" label="Alerts" />
    <FormField label="Volume" name="volume"><Slider v-model="state.volume" /></FormField>
    <FormField label="Code" name="code"><PinInput v-model="state.code" :length="3" /></FormField>
    <FieldGroup><Input v-model="state.site" aria-label="Site" /><Button label="Go" /></FieldGroup>
    <Button type="submit" label="Send" />
    <output>{{ state.email }}|{{ state.plan }}|{{ state.volume }}</output>
  </Form>
</template>
