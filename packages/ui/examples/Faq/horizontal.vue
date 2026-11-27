<script setup vapor lang="ts">
import { Faq, Link, type FaqItem } from '@weft/ui'

const items: FaqItem[] = [
  {
    label: 'Why is there no npm package to install?',
    content: 'The weft binary carries @weft/ui, @weft/vite, @weft/themes and the Go package so every part of an app stays in lockstep.',
  },
  {
    label: 'How do I update an existing app?',
    content: 'Run weft upgrade inside the project root to unpack the latest bundle and refresh the generated types.',
  },
]
</script>

<template>
  <Faq
    orientation="horizontal"
    title="Questions and answers"
    description="When wide, the heading sits in a column beside the accordion."
    :items="items"
  >
    <template #aside>
      Need more detail? Read the <Link to="/docs">documentation</Link>.
    </template>
  </Faq>
</template>
