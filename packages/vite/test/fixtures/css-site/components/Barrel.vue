<script setup vapor lang="ts">
import { ref } from 'vue'
import { Switch } from '@weft/ui'

const on = ref(false)
</script>

<template>
  <!-- An island that imports the @weft/ui index and draws nothing from it on the server. -->
  <div class="barrel"><Switch v-if="false" v-model="on" /></div>
</template>
