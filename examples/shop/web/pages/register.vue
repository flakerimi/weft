<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import RegisterForm from '../components/RegisterForm.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Sign up', description: 'Ask for a link that finishes signing up.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Sign up" description="Ask for a link that finishes signing up.">
      <Section>
        <Island when="load" :component="RegisterForm" />
      </Section>
    </Page>
  </div>
</template>
