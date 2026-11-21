<script setup vapor lang="ts">
import { Button, OverlayPanel } from '@weft/ui'
</script>

<template>
  <!-- The inside of Dialog, Sheet and Drawer, here on the page. -->
  <div style="display: flex; flex-direction: column; max-inline-size: 24rem; border: 1px solid; border-radius: 16px">
    <OverlayPanel title="Invite" description="They get an email with a link." title-id="invite-title" close>
      <template #body><p>Who should join?</p></template>
      <template #footer="{ close }"><Button label="Send" @click="close()" /></template>
    </OverlayPanel>
  </div>
</template>
