<script setup vapor lang="ts">
import { Page, useHead } from '@weft/ui'

useHead({ title: 'Not found — Weft' })
</script>

<template>
  <div data-weft-surface="landing">
    <Page title="Not found" description="Nothing lives at this address.">
      <p><a href="/">Back to weft.base.al</a></p>
    </Page>
  </div>
</template>
