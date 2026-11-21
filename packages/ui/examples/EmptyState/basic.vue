<script setup vapor lang="ts">
import { EmptyState, Link } from '@weft/ui'
</script>

<template>
  <EmptyState
    icon="user-plus"
    title="No customers yet"
    description="Customers you add, or who sign up, appear here."
    :level="2"
    :actions="[{ label: 'Add customer', icon: 'user-plus' }, { label: 'Import a CSV' }]"
  >
    <template #footer><Link to="/docs/customers">How customers work</Link></template>
  </EmptyState>
</template>
