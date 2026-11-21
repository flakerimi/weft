<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Popover } from '../../../src/index.ts'

// One popover whose trigger moves between a part of the page marked not to
// be prefetched and one that is not, as an example's would in a gallery.
const marked = ref(true)
defineExpose({ marked })
</script>

<template>
  <div :data-weft-no-prefetch="marked ? '' : undefined">
    <Popover title="Links"><Button class="trigger">Links</Button><template #content><a class="inside" href="/x">X</a></template></Popover>
  </div>
</template>
