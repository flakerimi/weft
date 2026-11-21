<script setup vapor lang="ts">
import { ref } from 'vue'
import { Stack, Switch } from '@weft/ui'

const wifi = ref(true)
const sync = ref(false)
const saving = ref(false)
function save(on: boolean) {
  sync.value = on
  saving.value = true
  setTimeout(() => (saving.value = false), 1200)
}
</script>

<template>
  <Stack :gap="1">
    <Switch v-model="wifi" label="Wi-Fi" description="Join known networks automatically." />
    <!-- Loading blocks toggling while the change is saved. -->
    <Switch :model-value="sync" label="Sync photos" :loading="saving" @update:model-value="save" />
  </Stack>
</template>
