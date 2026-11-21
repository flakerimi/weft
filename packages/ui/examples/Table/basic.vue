<script setup vapor lang="ts">
import { ref } from 'vue'
import { Badge, Table, type RowKey, type TableColumn } from '@weft/ui'

interface Invoice {
  id: number
  customer: string
  status: 'paid' | 'open' | 'overdue'
  amount: number
}

const rows: Invoice[] = [
  { id: 1041, customer: 'Ada Lovelace', status: 'paid', amount: 1250 },
  { id: 1042, customer: 'Grace Hopper', status: 'open', amount: 320.5 },
  { id: 1043, customer: 'Alan Turing', status: 'overdue', amount: 980 },
  { id: 1044, customer: 'Barbara Liskov', status: 'paid', amount: 45 },
  { id: 1045, customer: 'Linus Torvalds', status: 'open', amount: 610 },
  { id: 1046, customer: 'Margaret Hamilton', status: 'paid', amount: 2210 },
]
const columns: TableColumn<Invoice>[] = [
  { key: 'id', label: 'Invoice', sortable: true, format: (v) => `#${v}` },
  { key: 'customer', label: 'Customer', sortable: true, rowHeader: true },
  { key: 'status', label: 'Status' },
  { key: 'amount', label: 'Amount', sortable: true, align: 'end', format: (v) => `€${(v as number).toFixed(2)}` },
]
const tones = { paid: 'success', open: 'neutral', overdue: 'danger' } as const
const sort = ref<string | null>('-amount')
const selected = ref<RowKey[]>([])
</script>

<template>
  <Table v-model:sort="sort" v-model:selected="selected" :rows="rows" :columns="columns" caption="Invoices" selectable :per-page="5">
    <template #status-cell="{ row }">
      <Badge :label="row.status" :tone="tones[row.status]" variant="soft" />
    </template>
  </Table>
</template>
