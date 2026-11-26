<script setup vapor lang="ts">
import { useHead } from '@weft/ui'

useHead({ title: 'Not found', description: 'Nothing lives at this address.' })
</script>

<template>
  <div data-weft-surface="app">
    <Page title="Not found" description="Nothing lives at this address.">
      <Section>
        <Link to="/">Back to the front page</Link>
      </Section>
    </Page>
  </div>
</template>
