<script setup vapor lang="ts">
// Overlays drawn open, and a table drawn narrow, by the server: a page's
// pictures of an app (a landing page's hero), never hydrated.
import { Button, DropdownMenu, Table, WeftProvider, type MenuItem, type TableColumn } from '@weft/ui'

const items: MenuItem[][] = [[{ label: 'Open device' }, { label: 'Reboot' }], [{ label: 'Remove', color: 'danger' }]]
interface Device { id: number; name: string; port: string }
const rows: Device[] = [{ id: 1, name: 'Krasniqi A.', port: '1/2/4' }]
const columns: TableColumn<Device>[] = [{ key: 'name', label: 'Device', rowHeader: true }, { key: 'port', label: 'Port' }]
</script>

<template>
  <section id="pointer">
    <WeftProvider input="pointer">
      <DropdownMenu :items="items" :portal="false" default-open><Button>Pointer</Button></DropdownMenu>
    </WeftProvider>
  </section>
  <section id="touch">
    <WeftProvider input="touch">
      <DropdownMenu :items="items" :portal="false" default-open><Button>Touch</Button></DropdownMenu>
    </WeftProvider>
  </section>
  <section id="closed">
    <DropdownMenu :items="items" :portal="false"><Button>Closed</Button></DropdownMenu>
  </section>
  <section id="portal">
    <DropdownMenu :items="items" default-open><Button>Portal</Button></DropdownMenu>
  </section>
  <section id="narrow">
    <WeftProvider width="narrow"><Table :rows="rows" :columns="columns" caption="Devices" /></WeftProvider>
  </section>
  <section id="wide">
    <Table :rows="rows" :columns="columns" caption="Devices" />
  </section>
</template>
