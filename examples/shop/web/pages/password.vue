<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import PasswordForm from '../components/PasswordForm.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Choose a password', description: 'Set the password a mailed link is for.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Choose a password" description="Set the password a mailed link is for.">
      <Section>
        <Island when="load" :component="PasswordForm" />
      </Section>
    </Page>
  </div>
</template>
