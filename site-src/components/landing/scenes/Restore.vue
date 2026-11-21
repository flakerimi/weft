<script setup vapor lang="ts">
// A scene: restoring an OLT's configuration. A FileUpload to drop the backup
// on, the upload's Progress, and the Toast that says it is done, drawn in
// place (it stays: no time to run out, no bar).
import { FileUpload, Progress, Toast } from '@weft/ui'
</script>

<template>
  <div class="scene-restore">
    <FileUpload label="Drop a config backup" description="or choose a file, up to 2 MB" accept=".cfg" />
    <div class="scene-restore__file">
      <span>olt-north.cfg</span><span>72 %</span>
      <Progress :model-value="72" label="olt-north.cfg uploaded" size="sm" />
    </div>
    <Toast title="Backup from 08:00 restored" tone="success" icon="circle-check" :actions="[{ label: 'Undo' }]" :duration="0" :progress="false" :close="false" />
  </div>
</template>
