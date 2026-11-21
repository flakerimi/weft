<script setup vapor lang="ts">
import Footer from '../../../src/shell/Footer.vue'
import Header from '../../../src/shell/Header.vue'

defineProps<{ menu?: boolean }>()
</script>

<template>
  <div>
    <Header v-if="menu" title="Acme">
      <a href="/docs" class="center-link">Docs</a>
      <template #actions><button class="action">Sign in</button></template>
      <template #menu="{ close }"><a href="/docs" class="menu-link" @click.prevent="close()">Docs</a></template>
    </Header>
    <Header v-else title="Acme">
      <a href="/docs" class="center-link">Docs</a>
    </Header>
    <Footer>
      <template #leading><span class="copy">© Acme</span></template>
      <a href="/privacy">Privacy</a>
      <template #trailing><a href="/social">Social</a></template>
    </Footer>
  </div>
</template>
