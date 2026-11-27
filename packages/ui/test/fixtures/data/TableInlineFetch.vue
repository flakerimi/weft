<script setup vapor lang="ts">
import { ref } from 'vue'
import Table from '../../../src/data/Table.vue'
import type { TableColumn, TableQuery, TableResult } from '../../../src/data/table.ts'
import type { Person } from './TableCases.vue'

const COLUMNS: TableColumn<Person>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
]

const q = ref('')
const asked: (string | undefined)[] = []

async function list(query: TableQuery): Promise<TableResult<Person>> {
  asked.push(query.q)
  return { items: [], total: 0 }
}

defineExpose({ q, asked })
</script>

<template>
  <Table :columns="COLUMNS" :fetch="(query) => list(query)" :search="q" />
</template>
