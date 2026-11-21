<script setup vapor lang="ts">
import { Button, Stack, Tooltip } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2">
    <!-- No name of its own: the tooltip's text becomes its aria-label, so touch users hear it too. -->
    <Tooltip text="Delete file">
      <Button icon="trash" variant="ghost" tone="neutral" />
    </Tooltip>
    <Tooltip text="Saves without publishing" placement="bottom">
      <Button label="Save draft" variant="outline" tone="neutral" />
    </Tooltip>
  </Stack>
</template>
