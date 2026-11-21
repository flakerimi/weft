<script setup vapor lang="ts">
import { ref } from 'vue'
import Badge from '@weft/ui/basics/Badge.vue'
import Button from '@weft/ui/basics/Button.vue'
import Kbd from '@weft/ui/basics/Kbd.vue'

const count = ref(0)
</script>

<template>
  <div class="tally">
    <Button @click="count++">Add</Button>
    <Badge>{{ count }}</Badge>
    <!-- Drawn only in the browser, never on the server. -->
    <Kbd v-if="count > 100">!</Kbd>
  </div>
</template>

<style>
.tally { display: flex; gap: 4px; }
</style>
