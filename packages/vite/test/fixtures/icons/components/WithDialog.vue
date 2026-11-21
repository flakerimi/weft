<script setup vapor lang="ts">
// Imports Dialog and DropdownMenu, which draw their close cross and chevrons inline.
import Dialog from '@weft/ui/overlays/Dialog.vue'
import DropdownMenu from '@weft/ui/overlays/DropdownMenu.vue'
</script>

<template>
  <Dialog title="Hi"><template #body>Hello</template></Dialog>
  <DropdownMenu :items="[]" />
</template>
