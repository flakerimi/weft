<script setup vapor lang="ts">
import { ref } from 'vue'
import { Checkbox } from '@weft/ui'

const accepted = ref(false)
</script>

<template>
  <Checkbox v-model="accepted" label="I accept the terms" description="You can withdraw at any time." />
</template>
