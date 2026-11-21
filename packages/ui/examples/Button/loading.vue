<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Stack } from '@weft/ui'

const saving = ref(false)
function save() {
  saving.value = true
  setTimeout(() => (saving.value = false), 1500)
}
</script>

<template>
  <!-- Loading keeps the width and focus, and blocks further presses. -->
  <Stack direction="row" :gap="2">
    <Button label="Save changes" icon="save" :loading="saving" @click="save" />
    <Button label="Disabled" disabled />
  </Stack>
</template>
