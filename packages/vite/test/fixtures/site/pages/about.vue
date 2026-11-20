<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import Badge from '../components/Badge.vue'

useHead({ title: 'About <Weft>', description: 'A "quoted" & <b>page</b>', image: '/og.png?a=1&b=2' })
</script>

<template>
  <main>
    <h1>About</h1>
    <Badge>new</Badge>
  </main>
</template>
