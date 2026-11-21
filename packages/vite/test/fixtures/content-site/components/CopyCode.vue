<script setup vapor lang="ts">
const props = defineProps<{ code: string }>()
const copy = () => navigator.clipboard.writeText(props.code)
</script>

<template>
  <button type="button" class="copy-code" @click="copy">Copy</button>
</template>
