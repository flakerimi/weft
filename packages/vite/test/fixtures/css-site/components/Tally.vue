<script setup vapor lang="ts">
import { ref } from 'vue'
import { Badge, Button } from '@weft/ui'

const count = ref(0)
</script>

<template>
  <div class="tally">
    <Button @click="count++">Add</Button>
    <Badge>{{ count }}</Badge>
  </div>
</template>

<style>
.tally { display: flex; gap: 4px; }
</style>
