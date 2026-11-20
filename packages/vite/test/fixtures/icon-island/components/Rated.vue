<script setup vapor lang="ts">
import { ref } from 'vue'
import { Icon } from '@weft/ui'
const n = ref(1)
</script>

<template>
  <button type="button" @click="n++"><Icon name="star" :label="`${n} stars`" /></button>
</template>
