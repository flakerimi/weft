<script setup vapor lang="ts">
// A component taking a Lucide name in its `icon` prop, as Weft's do.
import { Icon } from '@weft/ui'
defineProps<{ icon: string }>()
</script>

<template><span class="tag"><Icon :name="icon" /></span></template>
