<script setup vapor lang="ts">
import { useTemplateRef } from 'vue'
import { Button, Stack, Stepper, type StepperItem } from '@weft/ui'

const stepper = useTemplateRef<{ next: () => void; prev: () => void; hasNext: boolean; hasPrev: boolean }>('stepper')
const items: StepperItem[] = [
  { title: 'Account', icon: 'user', content: 'Choose a name.' },
  { title: 'Team', icon: 'users', content: 'Invite your team.' },
  { title: 'Done', icon: 'package', content: 'All set.' },
]
</script>

<template>
  <Stack :gap="4">
    <Stepper ref="stepper" :items="items" orientation="vertical" />
    <Stack direction="row" :gap="2">
      <Button label="Back" variant="outline" tone="neutral" :disabled="!stepper?.hasPrev" @click="stepper?.prev()" />
      <Button label="Continue" :disabled="!stepper?.hasNext" @click="stepper?.next()" />
    </Stack>
  </Stack>
</template>
