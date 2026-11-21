<script setup vapor lang="ts">
// Northline on a phone: the same devices page, its bar with a menu Button,
// the title and an add Button, then the same device table, which a narrow
// touch screen draws as cards, its open menu a bottom sheet. It draws the
// devices a phone's screen has room for.
import { Button } from '@weft/ui'
import DeviceTable from './DeviceTable.vue'
</script>

<template>
  <div class="northline northline--phone">
    <div class="northline__bar">
      <Button icon="menu" variant="outline" tone="neutral" aria-label="Menu" />
      <p class="northline__title">Devices</p>
      <Button icon="plus" aria-label="Add device" />
    </div>
    <DeviceTable :count="3" />
  </div>
</template>
