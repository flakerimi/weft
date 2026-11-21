<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Dialog } from '@weft/ui'

// Opened from code; `dismissible: false` keeps it open on Escape and the backdrop.
const open = ref(false)
</script>

<template>
  <Button label="Show terms" @click="open = true" />
  <Dialog v-model:open="open" title="Terms of use" :dismissible="false" fullscreen>
    <template #body><p>Read these before going on.</p></template>
    <template #footer="{ close }"><Button label="I agree" @click="close()" /></template>
  </Dialog>
</template>
