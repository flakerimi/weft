<script setup vapor lang="ts">
import { Pagination, WeftProvider } from '@weft/ui'
</script>

<template>
  <main>
    <Pagination id="both" :total="200" :default-page="10" :sibling-count="2" show-edges />
    <WeftProvider width="narrow"><Pagination id="narrow" :total="200" :default-page="10" :sibling-count="2" show-edges /></WeftProvider>
    <WeftProvider width="wide"><Pagination id="wide" :total="200" :default-page="10" :sibling-count="2" show-edges /></WeftProvider>
  </main>
</template>
