<script setup vapor lang="ts">
import { ref } from 'vue'
import { Icon } from '@weft/ui'
import Tag from '../components/Tag.vue'
const on = ref(true)
// Names in data, found by their `icon:` key.
const items = [{ label: 'Home', icon: 'house' }]
</script>

<template>
  <main>
    <Icon name="trash" />
    <Tag icon="plus" />
    <Icon :name="on ? 'check' : 'x'" />
    <Icon v-for="item in items" :key="item.label" :name="item.icon" />
  </main>
</template>
