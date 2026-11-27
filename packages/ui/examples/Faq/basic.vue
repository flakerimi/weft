<script setup vapor lang="ts">
import { Faq, type FaqItem } from '@weft/ui'

const items: FaqItem[] = [
  {
    label: 'Does a page ship JavaScript by default?',
    content: 'No. Pages pre-render to HTML at build time and ship client JavaScript only for components wrapped in an Island.',
  },
  {
    label: 'Can I use an existing Go net/http middleware?',
    content: 'Yes. Warp builds on the standard library http.ServeMux and accepts standard http.Handler middleware.',
  },
  {
    label: 'How are closed answers found by browser search?',
    content: 'Each answer panel uses hidden="until-found", so the browser can match text inside a closed question and open it automatically.',
  },
]
</script>

<template>
  <Faq
    headline="FAQ"
    title="Common questions"
    description="Everything you need to know before building your first app."
    :items="items"
  />
</template>
