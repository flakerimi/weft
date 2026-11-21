<script setup vapor lang="ts">
import { ref, useTemplateRef } from 'vue'
import ControllableProbe from './ControllableProbe.vue'

const count = ref(5)
const open = ref(false)
const probe = useTemplateRef<{ bump(): void; toggle(): void }>('probe')
defineExpose({ count, open, bump: () => probe.value?.bump(), toggle: () => probe.value?.toggle() })
</script>

<template>
  <ControllableProbe ref="probe" v-model="count" v-model:open="open" />
  <ControllableProbe class="free" />
</template>
