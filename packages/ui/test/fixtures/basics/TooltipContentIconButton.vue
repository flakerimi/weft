<script setup vapor lang="ts">
// A Tooltip names its trigger, not what its content holds: an icon-only Button there has no name.
import Button from '../../../src/basics/Button.vue'
import Tooltip from '../../../src/overlays/Tooltip.vue'
</script>

<template>
  <Tooltip text="Add" default-open>
    <button type="button">Trigger</button>
    <template #content><Button icon="plus" /></template>
  </Tooltip>
</template>
