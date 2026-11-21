<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Dialog, FormField, Input } from '@weft/ui'

const name = ref('report.pdf')
</script>

<template>
  <Dialog title="Rename file" description="The new name shows everywhere the file is shared.">
    <Button label="Rename" variant="outline" tone="neutral" />
    <template #body>
      <FormField label="Name"><Input v-model="name" /></FormField>
    </template>
    <template #footer="{ close }">
      <Button label="Cancel" variant="ghost" tone="neutral" @click="close()" />
      <Button label="Save" @click="close()" />
    </template>
  </Dialog>
</template>
