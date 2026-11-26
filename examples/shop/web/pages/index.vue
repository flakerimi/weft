<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import ShopHeader from '../components/ShopHeader.vue'

useHead({ title: 'Shop', description: 'A small back office: the products a shop sells.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Shop" description="A small back office: the products a shop sells.">
      <Section>
        <Stack :gap="2">
          <Link to="/products">The products</Link>
          <Link to="/login">Sign in</Link>
        </Stack>
      </Section>
    </Page>
  </div>
</template>
