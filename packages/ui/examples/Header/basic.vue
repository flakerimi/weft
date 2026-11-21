<script setup vapor lang="ts">
import { Button, Header, Link, NavigationMenu, Stack, type NavItem } from '@weft/ui'

const items: NavItem[] = [
  { label: 'Guide', to: '/guide' },
  { label: 'Components', to: '/components' },
  { label: 'Blog', to: '/blog' },
]
</script>

<template>
  <Header title="Weft" :sticky="false">
    <NavigationMenu :items="items" />
    <template #actions><Button label="Sign in" variant="outline" tone="neutral" size="sm" /></template>
    <template #menu="{ close }">
      <Stack :gap="2">
        <Link v-for="item in items" :key="item.to" :to="item.to" standalone @click="close()">{{ item.label }}</Link>
      </Stack>
    </template>
  </Header>
</template>
