<script setup vapor lang="ts">
import { usePrePaint } from '@weft/ui'
import linked from '../styles/linked.css?url'
import generated from '/generated/sheet.css?url'
import '../styles/page.css'

usePrePaint('linked', `document.documentElement.dataset.linked=${JSON.stringify(linked)}`)
</script>

<template>
  <main class="page">
    <a id="linked" :href="linked">Linked</a>
    <a id="generated" :href="generated">Generated</a>
  </main>
</template>
