<script setup vapor lang="ts">
// An island that opens a Dialog holding a Button. Neither is drawn on the
// server: the dialog renders only in the browser, after hydration.
import Button from '@weft/ui/basics/Button.vue'
import Dialog from '@weft/ui/overlays/Dialog.vue'
</script>

<template>
  <div class="opener">
    <Dialog title="Rename">
      <button type="button">Rename</button>
      <template #footer="{ close }"><Button label="Done" @click="close()" /></template>
    </Dialog>
  </div>
</template>
