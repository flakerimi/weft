<script setup vapor lang="ts">
// An icon-only Button that a Tooltip names: its text becomes the aria-label.
import Button from '../../../src/basics/Button.vue'
import Tooltip from '../../../src/overlays/Tooltip.vue'
</script>

<template>
  <Tooltip text="Add"><Button icon="plus" /></Tooltip>
</template>
