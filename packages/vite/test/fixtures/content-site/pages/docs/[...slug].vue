<script lang="ts">
import { contentPaths } from 'virtual:weft/content'

export const getPaths = async () => contentPaths('docs')
</script>

<script setup vapor lang="ts">
import { Island, useHead } from '@weft/ui'
import { useContent } from 'virtual:weft/content'
import '../../../../../src/code.css'
import CopyCode from '../../components/CopyCode.vue'

const props = defineProps<{ slug: string }>()
const page = useContent('docs', props.slug)
useHead({ title: page.frontmatter.title, description: page.frontmatter.description })
</script>

<template>
  <main>
    <nav class="toc">
      <a v-for="entry in page.toc" :key="entry.id" :href="`#${entry.id}`">{{ entry.text }}</a>
    </nav>
    <template v-for="(part, i) in page.parts" :key="i">
      <div v-if="'code' in part" class="code">
        <div v-html="part.html" />
        <Island when="visible" :component="CopyCode" :props="{ code: part.code }" />
      </div>
      <div v-else class="prose" v-html="part.html" />
    </template>
    <footer>
      <a v-if="page.prev" class="prev" :href="page.prev.path">{{ page.prev.title }}</a>
      <a v-if="page.next" class="next" :href="page.next.path">{{ page.next.title }}</a>
    </footer>
  </main>
</template>
