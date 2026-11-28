<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import ForgotForm from '../components/ForgotForm.vue'

useHead({ title: 'Forgot your password', description: 'Ask for a link that sets a new password.' })
</script>

<template>
  <div data-weft-surface="app">
    <Page title="Forgot your password" description="Ask for a link that sets a new password.">
      <Section>
        <Island when="load" :component="ForgotForm" />
      </Section>
    </Page>
  </div>
</template>
