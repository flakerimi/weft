<script setup vapor lang="ts">
import { ref } from 'vue'
import { Stack, Toggle } from '@weft/ui'

const bold = ref(true)
const italic = ref(false)
const pinned = ref(false)
</script>

<template>
  <Stack direction="row" :gap="2">
    <Toggle v-model="bold" icon="bold" aria-label="Bold" />
    <Toggle v-model="italic" icon="italic" aria-label="Italic" />
    <Toggle v-model="pinned" label="Pinned" variant="outline" />
  </Stack>
</template>
