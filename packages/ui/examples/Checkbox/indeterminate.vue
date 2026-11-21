<script setup vapor lang="ts">
import { computed, ref } from 'vue'
import { Checkbox, Stack } from '@weft/ui'

const files = ref([
  { name: 'report.pdf', on: true },
  { name: 'notes.txt', on: false },
])
const all = computed({
  get: () => (files.value.every((f) => f.on) ? true : files.value.some((f) => f.on) ? 'indeterminate' : false),
  set: (v: boolean | 'indeterminate') => files.value.forEach((f) => (f.on = v === true)),
})
</script>

<template>
  <Stack :gap="1">
    <Checkbox v-model="all" label="All files" />
    <Checkbox v-for="f in files" :key="f.name" v-model="f.on" :label="f.name" />
  </Stack>
</template>
