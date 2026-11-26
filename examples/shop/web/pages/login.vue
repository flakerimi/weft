<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import LoginForm from '../components/LoginForm.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Sign in', description: 'Sign in to the shop.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Sign in">
      <Section>
        <Island when="load" :component="LoginForm" />
      </Section>
    </Page>
  </div>
</template>
