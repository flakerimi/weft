<script setup vapor lang="ts">
// Links two elements by an id from useId; the second appears only after a click,
// so its id comes from the browser's count, which must match the server's.
import { ref } from 'vue'
import { useId } from '@weft/ui'
const id = useId('field')
const open = ref(false)
</script>

<template>
  <div>
    <button type="button" :id="id" @click="open = true">label</button>
    <p v-if="open" class="late" :aria-labelledby="id">late</p>
  </div>
</template>
