<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import SetupForm from '../components/SetupForm.vue'

useHead({ title: 'Set up the owner', description: 'Choose the owner’s password.' })
</script>

<template>
  <div data-weft-surface="app">
    <Page title="Set up the owner" description="Choose the password of the owner, whose email is AUTH_OWNER_EMAIL in .env.">
      <Section>
        <Island when="load" :component="SetupForm" />
      </Section>
    </Page>
  </div>
</template>
