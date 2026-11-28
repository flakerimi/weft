<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import UsersPanel from '../components/UsersPanel.vue'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Users', description: 'Who has an account, and the invitations not yet taken.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Users" description="Who has an account, and the invitations not yet taken.">
      <Section>
        <Island when="load" :component="UsersPanel" />
      </Section>
    </Page>
  </div>
</template>
