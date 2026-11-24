<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import Hello from '../components/Hello.vue'

useHead({ title: 'Welcome', description: 'A Weft app: Go and Vue in one binary.' })

const notes = [
  { title: 'Pages', text: 'Each file in web/pages is a page, built ahead of time.' },
  { title: 'Islands', text: 'Only the parts you mark run in the browser.' },
  { title: 'Go', text: 'api/api.go answers /api; the binary serves both.' },
]
</script>

<template>
  <div data-weft-surface="landing">
    <Page title="Welcome" description="A Weft app: Go and Vue in one binary.">
      <Section title="What you have">
        <Grid>
          <Card v-for="note in notes" :key="note.title">
            <template #header>{{ note.title }}</template>
            {{ note.text }}
          </Card>
        </Grid>
      </Section>
      <Section title="From the API">
        <Island when="load" :component="Hello" />
      </Section>
    </Page>
  </div>
</template>
