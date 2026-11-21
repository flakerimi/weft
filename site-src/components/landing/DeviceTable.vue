<script setup vapor lang="ts">
// Northline's devices: one Table, its status a Badge and its last column a
// DropdownMenu per device, the opened device's drawn open in place
// (:portal="false"). The hero draws this same component twice: on a
// desktop it is a table with a floating menu, on a phone its cards with the
// menu as a bottom sheet; the components decide, nothing here does.
import { Badge, Button, DropdownMenu, Table } from '@weft/ui'
import { columns, deviceMenu, devices, opened, statusOf } from '../../landing/northline.ts'

const props = defineProps<{
  /** How many devices it draws, from the first: as many as the frame shows. */
  count?: number
}>()
const rows = devices.slice(0, props.count)
</script>

<template>
  <Table :rows="rows" :columns="columns" row-key="id" caption="Devices" hide-caption class="landing-devices">
    <template #port-cell="{ row }"><code>{{ row.port }}</code></template>
    <template #status-cell="{ row }"><Badge :label="statusOf[row.status].label" :tone="statusOf[row.status].tone" variant="soft" /></template>
    <template #actions-cell="{ row }">
      <DropdownMenu :items="deviceMenu(row)" :portal="false" :default-open="row.id === opened" placement="bottom-end">
        <Button icon="ellipsis" variant="ghost" tone="neutral" size="sm" :aria-label="`Actions for ${row.name}`" />
      </DropdownMenu>
    </template>
  </Table>
</template>
