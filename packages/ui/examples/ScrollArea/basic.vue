<script setup vapor lang="ts">
import { ScrollArea } from '@weft/ui'

const notes = Array.from({ length: 30 }, (_, i) => `Release 1.${30 - i}: fixes and small improvements.`)
</script>

<template>
  <ScrollArea label="Release notes" max-height="14rem" shadow>
    <ul>
      <li v-for="note in notes" :key="note">{{ note }}</li>
    </ul>
  </ScrollArea>
</template>
