<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, EntityDialog, useDateFields, type EntitySchema } from '@weft/ui'

interface Event {
  id: number
  title: string
  due: string | null
  starts: string
}

// The DatePicker and InputTime for the dialog's dates; without this it draws the browser's own inputs.
useDateFields()

const open = ref(false)
const schema: EntitySchema = {
  title: { kind: 'string', required: true },
  due: { kind: 'string', format: 'date', nullable: true },
  starts: { kind: 'string', format: 'datetime', required: true },
}

async function create(body: Omit<Event, 'id'>): Promise<Event> {
  return { id: 1, ...body }
}
</script>

<template>
  <Button label="New event" icon="plus" @click="open = true" />
  <EntityDialog
    v-model:open="open"
    name="event"
    :schema="schema"
    :fields="[
      { name: 'title' },
      { name: 'due', hint: 'A date alone, the same in every zone.' },
      { name: 'starts', hint: 'In your time zone; saved in UTC.' },
    ]"
    :create="create"
  />
</template>
