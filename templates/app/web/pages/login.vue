<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import LoginForm from '../components/LoginForm.vue'

useHead({ title: 'Sign in', description: 'Sign in to the app.' })
</script>

<template>
  <div data-weft-surface="app">
    <Page title="Sign in">
      <Section>
        <Island when="load" :component="LoginForm" />
      </Section>
    </Page>
  </div>
</template>
