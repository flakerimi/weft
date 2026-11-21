<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import OnDemand from '../components/OnDemand.vue'
import Picker from '../components/Picker.vue'
</script>

<template>
  <main>
    <Island when="load" :component="OnDemand" />
    <Island when="load" :component="Picker" />
  </main>
</template>
