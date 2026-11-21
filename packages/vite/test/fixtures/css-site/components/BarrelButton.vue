<script setup vapor lang="ts">
// An island that takes one component from the @weft/ui index, by name.
import { ref } from 'vue'
import { Button } from '@weft/ui'

const count = ref(0)
</script>

<template>
  <div><Button @click="count++">Added {{ count }}</Button></div>
</template>
