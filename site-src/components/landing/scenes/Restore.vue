<script setup vapor lang="ts">
// A scene: restoring an OLT's configuration. A FileUpload to drop the backup
// on; the upload, a Stack of its Icon, name and share over its Progress; and
// the Toast that says it is done, drawn in place with its Undo Button (it
// stays: no time to run out, no bar).
import { FileUpload, Icon, Progress, Stack, Toast } from '@weft/ui'
</script>

<template>
  <Stack :gap="4" class="scene-restore">
    <FileUpload label="Drop a config backup" description="or choose a file, up to 2 MB" accept=".cfg" />
    <Stack :gap="2">
      <Stack direction="row" :gap="2" align="center" justify="between">
        <Stack direction="row" :gap="2" align="center"><Icon name="file-cog" />olt-north.cfg</Stack>
        <span>72 %</span>
      </Stack>
      <Progress :model-value="72" label="olt-north.cfg uploaded" size="sm" />
    </Stack>
    <Toast title="Backup from 08:00 restored" tone="success" icon="circle-check" :actions="[{ label: 'Undo' }]" :duration="0" :progress="false" :close="false" />
  </Stack>
</template>
