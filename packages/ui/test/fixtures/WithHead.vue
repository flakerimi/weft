<script setup vapor lang="ts">
import { useHead } from '../../src/index.ts'

useHead({ title: 'Browser title', description: 'd', image: '/i.png' })
</script>

<template>
  <p>with head</p>
</template>
