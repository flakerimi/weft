<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import ProductsTable from '../../components/ProductsTable.vue'
import ShopHeader from '../../components/ShopHeader.vue'

useHead({ title: 'Products', description: 'The products the shop sells.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Products">
      <Section>
        <Island when="load" :component="ProductsTable" />
      </Section>
    </Page>
  </div>
</template>
