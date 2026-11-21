<script lang="ts">
import { contentPaths } from 'virtual:weft/content'

export const getPaths = async () => contentPaths('docs')
</script>

<script setup vapor lang="ts">
import { useContent } from 'virtual:weft/content'

const props = defineProps<{ slug: string }>()
const page = useContent('docs', props.slug)
</script>

<template>
  <main v-html="page.html" />
</template>
