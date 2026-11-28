<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import AccountPanel from '../components/AccountPanel.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Your account', description: 'Your name, password, sessions and API keys.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Your account" description="Your name, password, sessions and API keys.">
      <Section>
        <Island when="load" :component="AccountPanel" />
      </Section>
    </Page>
  </div>
</template>
