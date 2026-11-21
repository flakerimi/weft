<script setup vapor lang="ts">
import { Button, ConfirmDialog } from '@weft/ui'

// A promise keeps the dialog open, Delete loading, until the work is done.
function remove() {
  return new Promise((resolve) => setTimeout(resolve, 1000))
}
</script>

<template>
  <ConfirmDialog
    title="Delete this file?"
    description="It is removed for everyone it is shared with. This cannot be undone."
    confirm-label="Delete"
    tone="danger"
    @confirm="remove"
  >
    <Button label="Delete" icon="trash" tone="danger" variant="soft" />
  </ConfirmDialog>
</template>
