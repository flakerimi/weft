<script setup vapor lang="ts">
import { Island, Page } from '@weft/ui'
import Counter from './Counter.vue'
</script>

<template>
  <Page title="Static page, one interactive part">
    <Island when="visible" :component="Counter" :props="{ start: 0 }" />
  </Page>
</template>
