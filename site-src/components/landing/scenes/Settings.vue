<script setup vapor lang="ts">
// A scene: Northline's workspace settings, a Form of FormFields.
import { reactive } from 'vue'
import { Button, Form, FormField, Input, RadioGroup, Select, Stack, Switch } from '@weft/ui'

const state = reactive({ name: 'Northline Prishtina', region: 'fra', plan: 'team', alerts: true })
const regions = [
  { value: 'fra', label: 'Europe (Frankfurt)' },
  { value: 'ams', label: 'Europe (Amsterdam)' },
]
const plans = [
  { value: 'starter', label: 'Starter' },
  { value: 'team', label: 'Team' },
  { value: 'scale', label: 'Scale' },
]
</script>

<template>
  <Form :state="state" class="scene-settings">
    <FormField label="Workspace name" name="name"><Input v-model="state.name" autocomplete="organization" /></FormField>
    <FormField label="Region" name="region"><Select v-model="state.region" :items="regions" /></FormField>
    <FormField label="Plan" name="plan"><RadioGroup v-model="state.plan" :items="plans" orientation="horizontal" /></FormField>
    <Switch v-model="state.alerts" label="Email me when a device goes offline" />
    <Stack direction="row" :gap="2">
      <Button type="submit" label="Save changes" />
      <Button label="Cancel" variant="outline" tone="neutral" />
    </Stack>
  </Form>
</template>
