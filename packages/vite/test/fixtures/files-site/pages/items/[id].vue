<script lang="ts">
export const getPaths = async () => [{ id: 'a' }, { id: 'b' }]
</script>

<script setup vapor lang="ts">
import { useFile } from 'virtual:weft/files'

const props = defineProps<{ id: string }>()
useFile(`/items/${props.id}.json`, JSON.stringify({ id: props.id }))
useFile('/shared.txt', 'one text\n')
</script>

<template>
  <h1>{{ id }}</h1>
</template>
