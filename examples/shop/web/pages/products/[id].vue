<script lang="ts">
// Products are made after the build, so this page is built once, as a shell,
// and served for every /products/<id>; its island reads the id.
export const runtime = true
</script>

<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import ProductView from '../../components/ProductView.vue'
import ShopHeader from '../../components/ShopHeader.vue'

useHead({ title: 'Product', description: 'One product of the shop.' })
</script>

<template>
  <div data-weft-surface="app">
    <ShopHeader />
    <Page title="Product">
      <Section>
        <Island when="load" :component="ProductView" />
      </Section>
    </Page>
  </div>
</template>
