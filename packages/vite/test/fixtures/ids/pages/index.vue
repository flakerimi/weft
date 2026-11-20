<script setup vapor lang="ts">
import { Island, useId } from '@weft/ui'
import Labelled from '../components/Labelled.vue'
// The page's own ids count in the page's app, not in an island's.
const pageId = useId('page')
</script>

<template>
  <main :id="pageId">
    <Island when="load" :component="Labelled" />
    <Island when="load" :component="Labelled" />
  </main>
</template>
