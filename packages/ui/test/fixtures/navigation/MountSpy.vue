<script setup vapor lang="ts">
// Says when it mounts and unmounts, so a test can see a panel's content come and go.
import { onMounted, onUnmounted } from 'vue'

const props = defineProps<{ name: string; log: string[] }>()
onMounted(() => props.log.push(`+${props.name}`))
onUnmounted(() => props.log.push(`-${props.name}`))
</script>

<template>
  <input class="spy" :name="name" />
</template>
