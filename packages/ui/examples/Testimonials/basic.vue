<script setup vapor lang="ts">
import { Testimonials, type TestimonialItem } from '@weft/ui'

const items: TestimonialItem[] = [
  {
    quote: 'Deploying our admin tool went from three containers to one 18 MB binary on a tiny VM.',
    author: 'Ada Lovelace',
    role: 'CTO at Analytical Engine',
  },
  {
    quote: 'The forms use the exact validation rules our Go handlers declare. Drift between client and server disappeared.',
    author: 'Grace Hopper',
    role: 'Principal Engineer at Cobol Works',
  },
  {
    quote: 'Our docs and marketing pages ship zero JavaScript unless a page embeds an interactive island.',
    author: 'Margaret Hamilton',
    role: 'Director of Flight Software',
  },
]
</script>

<template>
  <Testimonials
    headline="In production"
    title="Teams shipping with Weft"
    :items="items"
  />
</template>
