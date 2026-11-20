<script setup vapor lang="ts">
import { ref } from 'vue'
import shared from './shared.module.css'

const n = ref(0)
</script>

<template>
  <button type="button" :class="[$style.tint, shared.wide, n > 0 && $style.on]" @click="n++">n is {{ n }}</button>
</template>

<style module>
.tint { color: rebeccapurple; }
.on { font-weight: 700; }
</style>
