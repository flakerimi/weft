<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Data from '../components/Data.vue'
</script>

<template>
  <div><Island when="load" :component="Data" /></div>
</template>
