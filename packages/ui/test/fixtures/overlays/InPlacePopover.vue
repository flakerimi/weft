<script setup vapor lang="ts">
import { Button, Popover, WeftProvider } from '../../../src/index.ts'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ defaultOpen?: boolean; input?: Input }>()
</script>

<template>
  <WeftProvider :input="props.input">
    <div class="before"><button class="elsewhere">Elsewhere</button></div>
    <div class="place">
      <Popover :portal="false" :default-open="defaultOpen" title="Filters">
        <Button class="trigger">Filters</Button>
        <template #content><button class="inside">Apply</button></template>
      </Popover>
    </div>
  </WeftProvider>
</template>
