<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Input, Stack } from '@weft/ui'

const query = ref('')
const amount = ref<string | number | null>(null)
</script>

<template>
  <Stack :gap="3">
    <Input v-model="query" icon="search" placeholder="Search invoices" aria-label="Search invoices" :loading="query.length > 2">
      <template #trailing>
        <Button v-if="query" icon="x" size="sm" variant="ghost" tone="neutral" aria-label="Clear" @click="query = ''" />
      </template>
    </Input>
    <Input v-model.number="amount" inputmode="decimal" aria-label="Amount">
      <template #leading>€</template>
      <template #trailing>EUR</template>
    </Input>
  </Stack>
</template>
