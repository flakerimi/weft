<script setup vapor lang="ts">
import { ref } from 'vue'
import { Card, CloseButton } from '@weft/ui'

const shown = ref(true)
</script>

<template>
  <Card v-if="shown">
    <template #actions><CloseButton label="Dismiss" @click="shown = false" /></template>
    A note you can dismiss.
  </Card>
</template>
