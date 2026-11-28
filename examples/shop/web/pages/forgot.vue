<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import ForgotForm from '../components/ForgotForm.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Forgot your password', description: 'Ask for a link that sets a new password.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Forgot your password" description="Ask for a link that sets a new password.">
      <Section>
        <Island when="load" :component="ForgotForm" />
      </Section>
    </Page>
  </div>
</template>
