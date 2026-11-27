<script setup vapor lang="ts">
// FormFields whose label slot holds a component that counts its setups,
// over a control that is no labelable element and over one that is.
import FormField from '../../../src/forms/FormField.vue'
import Input from '../../../src/forms/Input.vue'
import RadioGroup from '../../../src/forms/RadioGroup.vue'
import SetupCount from './SetupCount.vue'
</script>

<template>
  <div>
    <FormField class="plan" name="plan">
      <template #label>Plan <SetupCount name="plan" /></template>
      <RadioGroup :items="['free', 'pro']" />
    </FormField>
    <FormField class="email" name="email">
      <template #label>Email <SetupCount name="email" /></template>
      <Input />
    </FormField>
  </div>
</template>
