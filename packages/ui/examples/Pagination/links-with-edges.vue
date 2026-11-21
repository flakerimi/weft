<script setup vapor lang="ts">
import { Pagination } from '@weft/ui'

const to = (page: number) => `/orders?page=${page}`
</script>

<template>
  <Pagination :default-page="7" :total="400" :sibling-count="1" show-edges :to="to" />
</template>
