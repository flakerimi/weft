<script setup vapor lang="ts">
import { ref } from 'vue'
import { Pagination } from '@weft/ui'

const page = ref(1)
</script>

<template>
  <Pagination v-model:page="page" :total="240" :items-per-page="20" />
</template>
