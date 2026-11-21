<script setup vapor lang="ts">
// Northline on a desktop: a sidebar of NavigationMenu items beside the
// devices page, its Breadcrumb, title and actions (Buttons), three
// StatCards and the device table, a row's menu open.
import { Breadcrumb, Button, NavigationMenu, StatCard } from '@weft/ui'
import { nav } from '../../landing/northline.ts'
import DeviceTable from './DeviceTable.vue'
</script>

<template>
  <div class="northline">
    <aside class="northline__side">
      <p class="northline__brand"><span class="northline__logo" />Northline</p>
      <NavigationMenu :items="nav" orientation="vertical" label="Northline" />
    </aside>
    <div class="northline__main">
      <Breadcrumb :items="[{ label: 'Network' }, { label: 'Prishtina' }]" />
      <div class="northline__bar">
        <p class="northline__title">Devices</p>
        <Button label="Export" icon="download" variant="outline" tone="neutral" />
        <Button label="Add device" icon="plus" />
      </div>
      <div class="northline__stats">
        <StatCard label="Online" :value="1284" :delta="12" description="today" />
        <StatCard label="Offline" :value="17" :delta="3" invert description="today" />
        <StatCard label="Downstream" value="412 Mb/s" :delta="0.8" delta-unit="%" description="this hour" />
      </div>
      <DeviceTable />
    </div>
  </div>
</template>
