<script setup vapor lang="ts">
import '@weft/ui/styles/base.css'
import { Island, useHead } from '@weft/ui'
import Badge from '../components/Badge.vue'
import Lazy from '../components/Lazy.vue'

useHead({ title: 'About <Weft>', description: 'A "quoted" & <b>page</b>', image: '/og.png?a=1&b=2' })
</script>

<template>
  <main>
    <h1>About</h1>
    <Badge>new</Badge>
    <Island when="idle" :component="Lazy" />
  </main>
</template>
