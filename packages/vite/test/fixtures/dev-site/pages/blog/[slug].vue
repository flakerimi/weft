<script lang="ts">
import { readFileSync } from 'node:fs'

// Read on every call, so a post added to the file is a page on the next request.
export async function getPaths() {
  const posts = JSON.parse(readFileSync(new URL('../../content/posts.json', import.meta.url), 'utf8')) as string[]
  return posts.map((slug) => ({ slug }))
}
</script>

<script setup vapor lang="ts">
defineProps<{ slug: string }>()
</script>

<template>
  <main>
    <h1>Post {{ slug }}</h1>
  </main>
</template>
