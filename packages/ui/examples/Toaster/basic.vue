<script setup vapor lang="ts">
import { Button, Stack, Toaster, useToast } from '@weft/ui'

const toast = useToast()
function save() {
  toast.add({ title: 'Saved', description: 'Your draft is safe.', tone: 'success', icon: 'circle-check' })
}
function remove() {
  toast.add({ title: 'Message deleted', actions: [{ label: 'Undo', onClick: () => toast.add({ title: 'Restored' }) }] })
}
</script>

<template>
  <Stack direction="row" :gap="2">
    <Button label="Save" @click="save" />
    <Button label="Delete" variant="outline" tone="neutral" @click="remove" />
  </Stack>
  <!-- Once, in the app's layout. -->
  <Toaster position="bottom-end" />
</template>
