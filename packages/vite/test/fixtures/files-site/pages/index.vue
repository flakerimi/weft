<script setup vapor lang="ts">
// Writes /catalog.json beside the pages, and shows a highlighted line.
import { useFile } from 'virtual:weft/files'
import { code } from '../data.ts'

useFile('/catalog.json', '{"items":["a","b"]}\n')
// Every page may give the same file again, with the same bytes.
useFile('/shared.txt', 'one text\n')
</script>

<template>
  <main v-html="code" />
</template>
