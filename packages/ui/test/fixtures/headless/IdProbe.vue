<script setup vapor lang="ts">
import { useId } from '../../../src/headless/useId.ts'
const props = defineProps<{ prefix?: string }>()
const id = useId(props.prefix)
</script>

<template><span class="id" :id="id"></span></template>
