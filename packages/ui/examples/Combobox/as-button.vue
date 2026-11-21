<script setup vapor lang="ts">
import { ref } from 'vue'
import { Combobox, FormField } from '@weft/ui'

// Nuxt UI's SelectMenu: a button showing the choice, its list searched from a field at the top.
const timezone = ref('Europe/Belgrade')
const zones = ['Africa/Lagos', 'America/New_York', 'America/Sao_Paulo', 'Asia/Tokyo', 'Australia/Sydney', 'Europe/Belgrade', 'Europe/London', 'Pacific/Auckland']
</script>

<template>
  <FormField label="Time zone">
    <Combobox v-model="timezone" :items="zones" trigger="button" search-placeholder="Find a zone…" />
  </FormField>
</template>
