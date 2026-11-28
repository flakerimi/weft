<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import RegisterForm from '../components/RegisterForm.vue'

useHead({ title: 'Sign up', description: 'Ask for a link that finishes signing up.' })
</script>

<template>
  <div data-weft-surface="app">
    <Page title="Sign up" description="Ask for a link that finishes signing up.">
      <Section>
        <Island when="load" :component="RegisterForm" />
      </Section>
    </Page>
  </div>
</template>
