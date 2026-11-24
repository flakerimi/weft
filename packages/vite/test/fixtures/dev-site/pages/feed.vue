<script setup vapor lang="ts">
import { useFile } from 'virtual:weft/files'

useFile('/feed.json', '{"posts":["first"]}\n')
</script>

<template>
  <main>
    <h1>Feed</h1>
  </main>
</template>
